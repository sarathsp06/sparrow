from enum import StrEnum


class RegisterWebhookBodyTemplateMissingKey(StrEnum):
    ERROR = "error"
    VALUE_2 = ""
    ZERO = "zero"

    def __str__(self) -> str:
        return str(self.value)
