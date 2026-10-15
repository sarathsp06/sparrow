from enum import StrEnum


class RegisterWebhookBodyOnTransformError(StrEnum):
    FAIL = "fail"
    FALLBACK = "fallback"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
