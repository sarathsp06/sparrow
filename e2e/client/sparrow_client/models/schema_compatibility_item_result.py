from enum import StrEnum


class SchemaCompatibilityItemResult(StrEnum):
    BREAKING = "breaking"
    COMPATIBLE = "compatible"

    def __str__(self) -> str:
        return str(self.value)
