from enum import StrEnum


class ImportEventTypesInputBodyAcknowledgeItem(StrEnum):
    FORMAT_UNSUPPORTED = "format_unsupported"
    ITEMS_CHANGED = "items_changed"
    VERSION_DIFFERS = "version_differs"

    def __str__(self) -> str:
        return str(self.value)
