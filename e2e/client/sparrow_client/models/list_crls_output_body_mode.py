from enum import StrEnum


class ListCrlsOutputBodyMode(StrEnum):
    HARD = "hard"
    OFF = "off"
    SOFT = "soft"

    def __str__(self) -> str:
        return str(self.value)
