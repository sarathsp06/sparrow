# frozen_string_literal: true

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
#   webhook resource's +signing_public_key+ field. Requires OpenSSL 1.1.1+
#   with Ed25519 support.
#
# Copy this file into your project or vendor it as-is; it depends only on
# the Ruby stdlib (+openssl+).
#
# Usage (Rails):
#
#     require_relative "sparrow_verify"
#
#     class WebhooksController < ApplicationController
#       skip_before_action :verify_authenticity_token
#
#       def receive
#         # IMPORTANT: use request.raw_post, not params — the raw body must
#         # match what was signed.
#         SparrowVerify.verify_hmac(request.raw_post, request.headers, secret)
#         head :ok
#       rescue SparrowVerify::SignatureVerificationError
#         head :unauthorized
#       end
#     end
#
# Usage (Sinatra):
#
#     require_relative "sparrow_verify"
#
#     post "/webhook" do
#       payload = request.body.read   # raw body
#       SparrowVerify.verify_hmac(payload, request.env, ENV["WEBHOOK_SECRET"])
#       status 200
#     rescue SparrowVerify::SignatureVerificationError
#       halt 401
#     end

require "openssl"

module SparrowVerify
  DEFAULT_TOLERANCE_SECONDS = 5 * 60

  class SignatureVerificationError < StandardError; end

  # Verify the "v1," (HMAC-SHA256) signature. Raises on failure.
  #
  # +payload+ must be the raw request body string, exactly as received.
  # +headers+ is a Hash (or any object responding to +each+) with string
  # keys, looked up case-insensitively. Rack-style names (HTTP_WEBHOOK_ID, as
  # in request.env or Rails' request.headers) are accepted too.
  # +secret+ is the webhook secret, with or without the "whsec_" prefix.
  # +now+ overrides the clock (Unix seconds integer or float), e.g. in tests.
  def self.verify_hmac(payload, headers, secret,
                       tolerance_seconds: DEFAULT_TOLERANCE_SECONDS,
                       now: nil)
    msg_id, timestamp, sig_header = required_headers(headers)
    check_timestamp(timestamp, tolerance_seconds, now)

    if secret.start_with?("whsec_")
      encoded = secret[6..]
      begin
        key = strict_base64_decode(encoded)
      rescue ArgumentError => e
        raise SignatureVerificationError, "invalid whsec_ secret: #{e.message}"
      end
    else
      key = secret.b
    end
    raise SignatureVerificationError, "webhook secret is empty" if key.empty?

    message = "#{msg_id}.#{timestamp}.".b + payload.b
    expected = OpenSSL::HMAC.digest("SHA256", key, message)

    decode_signatures(sig_header, "v1,").each do |candidate|
      if fixed_length_secure_compare(candidate, expected)
        return
      end
    end
    raise SignatureVerificationError, "no matching v1 (HMAC-SHA256) signature"
  end

  # Verify the "v1a," (Ed25519) signature. Raises on failure.
  #
  # +payload+ must be the raw request body string, exactly as received.
  # +public_key_hex+ is the hex-encoded key from the webhook resource's
  # +signing_public_key+ field.
  # +now+ overrides the clock (Unix seconds), e.g. in tests.
  def self.verify_ed25519(payload, headers, public_key_hex,
                          tolerance_seconds: DEFAULT_TOLERANCE_SECONDS,
                          now: nil)
    msg_id, timestamp, sig_header = required_headers(headers)
    check_timestamp(timestamp, tolerance_seconds, now)

    raw_key = begin
      [public_key_hex].pack("H*")
    rescue ArgumentError => e
      raise SignatureVerificationError, "invalid hex public key: #{e.message}"
    end
    # Validate hex round-trips (rejects odd-length or non-hex chars that pack silently truncates)
    unless raw_key.bytesize == 32 && raw_key.unpack1("H*") == public_key_hex.downcase
      raise SignatureVerificationError, "invalid hex public key: must be exactly 32 bytes"
    end

    # Build DER SubjectPublicKeyInfo for Ed25519:
    #   SEQUENCE { SEQUENCE { OID 1.3.101.112 }, BIT STRING <key> }
    der_prefix = ["302a300506032b6570032100"].pack("H*")
    der = der_prefix + raw_key
    pkey = begin
      OpenSSL::PKey.read(der)
    rescue OpenSSL::PKey::PKeyError => e
      raise SignatureVerificationError, "Ed25519 not supported by this OpenSSL: #{e.message}"
    end

    message = "#{msg_id}.#{timestamp}.".b + payload.b

    decode_signatures(sig_header, "v1a,").each do |candidate|
      begin
        # Ed25519 uses nil for the digest algorithm
        if pkey.verify(nil, candidate, message)
          return
        end
      rescue OpenSSL::PKey::PKeyError
        next
      end
    end
    raise SignatureVerificationError, "no matching v1a (Ed25519) signature"
  end

  # --- private helpers ---

  def self.required_headers(headers)
    lowered = {}
    if headers.respond_to?(:each)
      headers.each do |k, v|
        name = k.to_s.downcase
        # Rack env / Rails request.headers expose HTTP_WEBHOOK_ID-style keys.
        name = name.delete_prefix("http_").tr("_", "-") if name.start_with?("http_")
        lowered[name] = v.is_a?(Array) ? v.first.to_s : v.to_s
      end
    end

    msg_id = lowered["webhook-id"]
    timestamp = lowered["webhook-timestamp"]
    sig_header = lowered["webhook-signature"]

    if msg_id.nil? || msg_id.empty? ||
       timestamp.nil? || timestamp.empty? ||
       sig_header.nil? || sig_header.empty?
      raise SignatureVerificationError, "missing webhook-id, webhook-timestamp, or webhook-signature header"
    end

    [msg_id, timestamp, sig_header]
  end
  private_class_method :required_headers

  def self.check_timestamp(timestamp, tolerance_seconds, now)
    unless timestamp.match?(/\A-?\d+\z/)
      raise SignatureVerificationError, "invalid webhook-timestamp: #{timestamp.inspect}"
    end
    seconds = timestamp.to_i
    current = now || Time.now.to_f
    if (current - seconds).abs > tolerance_seconds
      raise SignatureVerificationError, "webhook-timestamp outside tolerance (possible replay)"
    end
  end
  private_class_method :check_timestamp

  def self.decode_signatures(sig_header, prefix)
    results = []
    sig_header.split(/\s+/).each do |part|
      next if part.empty?
      next unless part.start_with?(prefix)
      encoded = part[prefix.length..]
      begin
        decoded = strict_base64_decode(encoded)
        results << decoded unless decoded.empty?
      rescue ArgumentError
        next
      end
    end
    results
  end
  private_class_method :decode_signatures

  # Strict base64 decode (rejects non-base64 characters, requires padding).
  # Uses unpack1("m0") instead of the base64 gem (removed from default gems in Ruby 3.4).
  def self.strict_base64_decode(str)
    # Validate: only A-Za-z0-9+/ and trailing = padding, length multiple of 4
    unless str.match?(%r{\A[A-Za-z0-9+/]*={0,2}\z}) && (str.length % 4).zero?
      raise ArgumentError, "invalid base64"
    end
    str.unpack1("m0")
  end
  private_class_method :strict_base64_decode

  # Constant-time comparison that doesn't leak the expected length when
  # the candidate length differs.
  def self.fixed_length_secure_compare(a, b)
    return false unless a.bytesize == b.bytesize
    if OpenSSL.respond_to?(:fixed_length_secure_compare)
      # Ruby >= 2.7 / openssl gem >= 2.2
      OpenSSL.fixed_length_secure_compare(a, b)
    else
      # Fallback: constant-time XOR comparison
      a_bytes = a.bytes
      b_bytes = b.bytes
      result = 0
      a_bytes.each_index { |i| result |= a_bytes[i] ^ b_bytes[i] }
      result.zero?
    end
  end
  private_class_method :fixed_length_secure_compare
end
