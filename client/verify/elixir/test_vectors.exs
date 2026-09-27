# Test runner for Sparrow signature verification vectors.
# Run: elixir test_vectors.exs   (from client/verify/elixir/)

Code.require_file("sparrow_verify.ex", __DIR__)

# Resolve vectors.json relative to this script.
script_dir = __DIR__
vectors_path = Path.join([script_dir, "..", "..", "..", "pkg", "signature", "testdata", "vectors.json"])

data = File.read!(vectors_path) |> JSON.decode!()
now = data["now"]
tolerance = data["tolerance_seconds"]
cases = data["cases"]

results =
  Enum.map(cases, fn c ->
    payload = Base.decode64!(c["payload_b64"])
    headers = c["headers"]

    result =
      case c["scheme"] do
        "hmac" ->
          SparrowVerify.verify_hmac(payload, headers, c["key"], tolerance, now)

        "ed25519" ->
          SparrowVerify.verify_ed25519(payload, headers, c["key"], tolerance, now)
      end

    got_valid = match?({:ok, :verified}, result)
    expected = c["valid"]
    status = if got_valid == expected, do: :pass, else: :fail

    case status do
      :pass ->
        IO.puts("PASS: #{c["name"]}")

      :fail ->
        err_msg =
          case result do
            {:error, reason} -> reason
            {:ok, :verified} -> "(verified)"
          end

        IO.puts(
          "FAIL: #{c["name"]} (expected valid=#{expected}, got valid=#{got_valid}, err=#{err_msg})"
        )
    end

    status
  end)

pass_count = Enum.count(results, &(&1 == :pass))
fail_count = Enum.count(results, &(&1 == :fail))

IO.puts("\n#{pass_count} passed, #{fail_count} failed out of #{length(results)} cases")

if fail_count > 0 do
  System.halt(1)
end
