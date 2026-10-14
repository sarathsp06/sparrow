from enum import StrEnum


class ImportItemResultActiveChange(StrEnum):
    DEACTIVATES = "deactivates"
    REACTIVATES = "reactivates"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
