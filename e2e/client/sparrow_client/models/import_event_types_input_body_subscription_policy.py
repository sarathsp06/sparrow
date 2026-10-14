from enum import StrEnum


class ImportEventTypesInputBodySubscriptionPolicy(StrEnum):
    KEEP_ACTIVE = "keep_active"
    PAUSE = "pause"

    def __str__(self) -> str:
        return str(self.value)
