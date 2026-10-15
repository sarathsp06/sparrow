from enum import StrEnum


class DraftTemplateOutputBodySampleSource(StrEnum):
    PROVIDED = "provided"
    REGISTERED = "registered"

    def __str__(self) -> str:
        return str(self.value)
