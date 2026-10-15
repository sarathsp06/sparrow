from enum import StrEnum


class EventTypeBundleApiVersion(StrEnum):
    SPARROWV1 = "sparrow/v1"

    def __str__(self) -> str:
        return str(self.value)
