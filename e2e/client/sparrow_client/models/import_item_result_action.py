from enum import StrEnum


class ImportItemResultAction(StrEnum):
    CREATED = "created"
    NEW_VERSION = "new_version"
    UNCHANGED = "unchanged"
    UPDATED = "updated"

    def __str__(self) -> str:
        return str(self.value)
