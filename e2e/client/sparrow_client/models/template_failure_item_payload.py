from enum import StrEnum


class TemplateFailureItemPayload(StrEnum):
    REQUIRED_ONLY = "required_only"
    SAMPLE = "sample"

    def __str__(self) -> str:
        return str(self.value)
