from enum import StrEnum


class EventTypeBundleKind(StrEnum):
    EVENTTYPELIST = "EventTypeList"

    def __str__(self) -> str:
        return str(self.value)
