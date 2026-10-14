from enum import StrEnum


class ListDeliveriesGlobalStatus(StrEnum):
    EXPIRED = "expired"
    FAILED = "failed"
    PAUSED = "paused"
    PENDING = "pending"
    RETRYING = "retrying"
    SENDING = "sending"
    SUCCESS = "success"

    def __str__(self) -> str:
        return str(self.value)
