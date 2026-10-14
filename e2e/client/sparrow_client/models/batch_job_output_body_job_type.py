from enum import StrEnum


class BatchJobOutputBodyJobType(StrEnum):
    DELIVERY_RETRY = "delivery_retry"
    EVENT_REPUSH = "event_repush"

    def __str__(self) -> str:
        return str(self.value)
