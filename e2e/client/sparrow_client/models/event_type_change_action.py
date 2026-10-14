from enum import StrEnum


class EventTypeChangeAction(StrEnum):
    CREATED = "created"
    NEW_VERSION = "new_version"
    UNCHANGED = "unchanged"
    UPDATED = "updated"

    def __str__(self) -> str:
        return str(self.value)
