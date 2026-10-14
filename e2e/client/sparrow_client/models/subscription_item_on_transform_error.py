from enum import StrEnum


class SubscriptionItemOnTransformError(StrEnum):
    FAIL = "fail"
    FALLBACK = "fallback"

    def __str__(self) -> str:
        return str(self.value)
