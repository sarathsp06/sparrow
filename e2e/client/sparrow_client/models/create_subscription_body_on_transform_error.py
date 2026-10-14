from enum import StrEnum


class CreateSubscriptionBodyOnTransformError(StrEnum):
    FAIL = "fail"
    FALLBACK = "fallback"

    def __str__(self) -> str:
        return str(self.value)
