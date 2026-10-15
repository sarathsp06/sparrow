from enum import StrEnum


class ImportItemResultActiveChange(StrEnum):
    DEACTIVATES = "deactivates"
    REACTIVATES = "reactivates"

    def __str__(self) -> str:
        return str(self.value)
