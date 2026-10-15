from enum import StrEnum


class EventTypeChangeActiveChange(StrEnum):
    DEACTIVATES = "deactivates"
    REACTIVATES = "reactivates"

    def __str__(self) -> str:
        return str(self.value)
