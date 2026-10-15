from enum import StrEnum


class RegisterWebhookBodyTemplateMissingKey(StrEnum):
    ERROR = "error"
    ZERO = "zero"

    def __str__(self) -> str:
        return str(self.value)
