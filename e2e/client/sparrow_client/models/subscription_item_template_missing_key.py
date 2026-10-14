from enum import StrEnum


class SubscriptionItemTemplateMissingKey(StrEnum):
    ERROR = "error"
    ZERO = "zero"

    def __str__(self) -> str:
        return str(self.value)
