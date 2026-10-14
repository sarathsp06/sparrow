from enum import StrEnum


class ImportEventTypesInputBodyKind(StrEnum):
    EVENTTYPELIST = "EventTypeList"

    def __str__(self) -> str:
        return str(self.value)
