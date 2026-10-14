from enum import StrEnum


class StampCheckItemStatus(StrEnum):
    UNSIGNED = "unsigned"
    VALID = "valid"
    WARNING = "warning"

    def __str__(self) -> str:
        return str(self.value)
