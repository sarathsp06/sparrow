from enum import StrEnum


class ImportEventTypesInputBodyApiVersion(StrEnum):
    SPARROWV1 = "sparrow/v1"

    def __str__(self) -> str:
        return str(self.value)
