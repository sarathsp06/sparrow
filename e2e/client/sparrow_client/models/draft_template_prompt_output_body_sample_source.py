from enum import StrEnum


class DraftTemplatePromptOutputBodySampleSource(StrEnum):
    PROVIDED = "provided"
    REGISTERED = "registered"

    def __str__(self) -> str:
        return str(self.value)
