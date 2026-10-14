from enum import StrEnum


class TemplateVersionItemSource(StrEnum):
    AI_DRAFT = "ai_draft"
    MANUAL = "manual"

    def __str__(self) -> str:
        return str(self.value)
