from enum import StrEnum


class RegisterWebhookBodyOnTransformError(StrEnum):
    FAIL = "fail"
    FALLBACK = "fallback"

    def __str__(self) -> str:
        return str(self.value)
