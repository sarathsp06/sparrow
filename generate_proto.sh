#!/bin/bash

# Generate protobuf files for the webhook service
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/webhook.proto

echo "Protobuf files generated successfully!"