# frozen_string_literal: true

# Run the shared signature vectors (pkg/signature/testdata/vectors.json).
#
#     ruby client/verify/ruby/test_vectors.rb

require "json"

HERE = File.expand_path(__dir__)
$LOAD_PATH.unshift(HERE)
require "sparrow_verify"

VECTORS = File.join(HERE, "..", "..", "..", "pkg", "signature", "testdata", "vectors.json")

data = JSON.parse(File.read(VECTORS))
now = data["now"]
tolerance = data["tolerance_seconds"]

passed = 0
failed = 0

# Every case runs twice: with plain header names, and as a Rack env
# (HTTP_WEBHOOK_ID), which is what Rails' request.headers and Sinatra's
# request.env hand to the helper.
rack = ->(h) { h.to_h { |k, v| ["HTTP_" + k.upcase.tr("-", "_"), v] } }
cases = data["cases"].flat_map do |c|
  [c, c.merge("name" => "#{c["name"]} (rack env)", "headers" => rack.call(c["headers"]))]
end

cases.each do |c|
  payload = c["payload_b64"].unpack1("m0")

  ok = begin
    if c["scheme"] == "hmac"
      SparrowVerify.verify_hmac(payload, c["headers"], c["key"],
                                tolerance_seconds: tolerance, now: now)
    else
      SparrowVerify.verify_ed25519(payload, c["headers"], c["key"],
                                   tolerance_seconds: tolerance, now: now)
    end
    true
  rescue SparrowVerify::SignatureVerificationError
    false
  end

  if ok == c["valid"]
    puts "PASS  #{c["name"]}"
    passed += 1
  else
    puts "FAIL  #{c["name"]} (expected valid=#{c["valid"]}, got #{ok})"
    failed += 1
  end
end

puts
puts "#{passed + failed} cases: #{passed} passed, #{failed} failed"
exit(1) if failed > 0
