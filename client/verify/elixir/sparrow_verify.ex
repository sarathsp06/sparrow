# Verify Sparrow webhook delivery signatures (Standard Webhooks format).
#
# Every delivery carries three headers:
#
#     webhook-id:        msg_<delivery-id>
#     webhook-timestamp: Unix seconds
#     webhook-signature: space-delimited signatures, e.g. "v1,<base64> v1a,<base64>"
#
# The signed message is "{webhook-id}.{webhook-timestamp}.{raw body}".
#
# - "v1,"  is HMAC-SHA256 keyed with the webhook secret. Secrets in Standard
#   Webhooks format ("whsec_" + base64) are decoded before use; any other
#   secret is used as raw bytes.
# - "v1a," is Ed25519, verified with the hex-encoded public key from the
#   webhook resource's `signing_public_key` field.
#
# Both verifiers reject deliveries whose webhook-timestamp is more than
# `tolerance` away from the current time, to prevent replay.
#
# Uses only :crypto and Base (stdlib) — no external dependencies.
# Copy this file into your project; it has no dependency on the Sparrow client.
#
# Usage (Phoenix/Plug):
#
#     # In your endpoint, use a custom body reader to capture the raw body:
#     plug Plug.Parsers,
#       parsers: [:json],
#       pass: ["application/json"],
#       body_reader: {MyApp.CacheBodyReader, :read_body, []}
#
#     # In your controller:
#     def webhook(conn, _params) do
#       raw_body = conn.assigns[:raw_body]
#       headers = Enum.into(conn.req_headers, %{})
#
#       case SparrowVerify.verify_hmac(raw_body, headers, secret) do
#         {:ok, :verified} -> send_resp(conn, 200, "ok")
#         {:error, reason} -> send_resp(conn, 401, reason)
#       end
#     end

defmodule SparrowVerify do
  @moduledoc """
  Verify Sparrow webhook delivery signatures (Standard Webhooks format).
  """

  @default_tolerance_seconds 300

  @type headers :: %{String.t() => String.t()} | [{String.t(), String.t()}]

  # ── HMAC-SHA256 ("v1,") ──────────────────────────────────────────────

  @doc """
  Verify the "v1," (HMAC-SHA256) signature. Returns `{:ok, :verified}` or
  `{:error, reason}`.

  `payload` must be the raw request body binary, exactly as received.
  `secret` is the webhook secret, with or without the "whsec_" prefix.
  `now` overrides the clock (Unix seconds); defaults to `System.os_time(:second)`.
  """
  @spec verify_hmac(binary(), headers(), String.t(), non_neg_integer(), integer() | nil) ::
          {:ok, :verified} | {:error, String.t()}
  def verify_hmac(payload, headers, secret, tolerance \\ @default_tolerance_seconds, now \\ nil) do
    with {:ok, {msg_id, timestamp, sigs}} <- parse_headers(headers, tolerance, now),
         {:ok, key} <- decode_secret(secret) do
      message = "#{msg_id}.#{timestamp}.#{payload}"
      expected = :crypto.mac(:hmac, :sha256, key, message)

      found =
        sigs
        |> decode_signatures("v1,")
        |> Enum.any?(fn candidate ->
          byte_size(candidate) == byte_size(expected) and
            :crypto.hash_equals(candidate, expected)
        end)

      if found, do: {:ok, :verified}, else: {:error, "no matching v1 (HMAC-SHA256) signature"}
    end
  end

  @doc """
  Like `verify_hmac/5` but raises on failure.
  """
  @spec verify_hmac!(binary(), headers(), String.t(), non_neg_integer(), integer() | nil) :: :ok
  def verify_hmac!(payload, headers, secret, tolerance \\ @default_tolerance_seconds, now \\ nil) do
    case verify_hmac(payload, headers, secret, tolerance, now) do
      {:ok, :verified} -> :ok
      {:error, reason} -> raise ArgumentError, reason
    end
  end

  # ── Ed25519 ("v1a,") ─────────────────────────────────────────────────

  @doc """
  Verify the "v1a," (Ed25519) signature. Returns `{:ok, :verified}` or
  `{:error, reason}`.

  `payload` must be the raw request body binary, exactly as received.
  `public_key_hex` is the hex-encoded key from the webhook resource's
  `signing_public_key` field. `now` overrides the clock (Unix seconds).
  """
  @spec verify_ed25519(binary(), headers(), String.t(), non_neg_integer(), integer() | nil) ::
          {:ok, :verified} | {:error, String.t()}
  def verify_ed25519(
        payload,
        headers,
        public_key_hex,
        tolerance \\ @default_tolerance_seconds,
        now \\ nil
      ) do
    with {:ok, {msg_id, timestamp, sigs}} <- parse_headers(headers, tolerance, now),
         {:ok, pub_key} <- decode_public_key(public_key_hex) do
      message = "#{msg_id}.#{timestamp}.#{payload}"

      found =
        sigs
        |> decode_signatures("v1a,")
        |> Enum.any?(fn candidate ->
          try do
            :crypto.verify(:eddsa, :none, message, candidate, [pub_key, :ed25519])
          rescue
            _ -> false
          end
        end)

      if found, do: {:ok, :verified}, else: {:error, "no matching v1a (Ed25519) signature"}
    end
  end

  @doc """
  Like `verify_ed25519/5` but raises on failure.
  """
  @spec verify_ed25519!(binary(), headers(), String.t(), non_neg_integer(), integer() | nil) :: :ok
  def verify_ed25519!(
        payload,
        headers,
        public_key_hex,
        tolerance \\ @default_tolerance_seconds,
        now \\ nil
      ) do
    case verify_ed25519(payload, headers, public_key_hex, tolerance, now) do
      {:ok, :verified} -> :ok
      {:error, reason} -> raise ArgumentError, reason
    end
  end

  # ── Internal helpers ──────────────────────────────────────────────────

  defp parse_headers(headers, tolerance, now) do
    lowered =
      headers
      |> Enum.into(%{}, fn {k, v} -> {String.downcase(to_string(k)), to_string(v)} end)

    with {:ok, msg_id} <- fetch_header(lowered, "webhook-id"),
         {:ok, timestamp} <- fetch_header(lowered, "webhook-timestamp"),
         {:ok, sig_header} <- fetch_header(lowered, "webhook-signature"),
         :ok <- validate_timestamp(timestamp, tolerance, now) do
      sigs =
        sig_header
        |> String.split(~r/\s+/, trim: true)

      {:ok, {msg_id, timestamp, sigs}}
    end
  end

  defp fetch_header(lowered, name) do
    case Map.fetch(lowered, name) do
      {:ok, val} when val != "" -> {:ok, val}
      _ -> {:error, "missing header: #{name}"}
    end
  end

  defp validate_timestamp(timestamp, tolerance, now) do
    digits = case timestamp do
      "-" <> rest -> rest
      other -> other
    end

    if digits == "" or not match?(<<_::binary>>, digits) or
         not Enum.all?(:binary.bin_to_list(digits), &(&1 >= ?0 and &1 <= ?9)) do
      {:error, "invalid webhook-timestamp: #{timestamp}"}
    else
      seconds = String.to_integer(timestamp)
      current = now || System.os_time(:second)
      skew = abs(current - seconds)

      if skew > tolerance do
        {:error, "webhook-timestamp outside tolerance (possible replay)"}
      else
        :ok
      end
    end
  end

  defp decode_secret(secret) do
    case secret do
      "whsec_" <> encoded ->
        case Base.decode64(encoded) do
          {:ok, <<>>} -> {:error, "webhook secret is empty"}
          {:ok, key} -> {:ok, key}
          :error -> {:error, "invalid whsec_ secret"}
        end

      "" ->
        {:error, "webhook secret is empty"}

      raw ->
        {:ok, raw}
    end
  end

  defp decode_public_key(hex) do
    case safe_hex_decode(hex) do
      {:ok, key} when byte_size(key) == 32 -> {:ok, key}
      {:ok, _} -> {:error, "invalid hex public key: must be 32 bytes"}
      :error -> {:error, "invalid hex public key"}
    end
  end

  defp safe_hex_decode(hex) when rem(byte_size(hex), 2) != 0, do: :error
  defp safe_hex_decode(hex) do
    try do
      {:ok, Base.decode16!(hex, case: :mixed)}
    rescue
      _ -> :error
    end
  end

  defp decode_signatures(sigs, prefix) do
    prefix_len = byte_size(prefix)

    sigs
    |> Enum.flat_map(fn part ->
      case part do
        <<^prefix::binary-size(prefix_len), encoded::binary>> ->
          case Base.decode64(encoded) do
            {:ok, decoded} -> [decoded]
            :error -> []
          end

        _ ->
          []
      end
    end)
  end
end
