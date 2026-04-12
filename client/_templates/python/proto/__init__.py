# Re-export generated protobuf and gRPC stubs for convenient imports.
#
# Usage:
#   from sparrow_webhooks.proto import webhook_pb2, webhook_pb2_grpc
#
#   channel = grpc.insecure_channel("localhost:50051")
#   stub = webhook_pb2_grpc.WebhookServiceStub(channel)

from . import webhook_pb2
from . import webhook_pb2_grpc

__all__ = ["webhook_pb2", "webhook_pb2_grpc"]
