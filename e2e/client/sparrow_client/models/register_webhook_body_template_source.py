from enum import StrEnum


class RegisterWebhookBodyTemplateSource(StrEnum):
    AI_DRAFT = "ai_draft"
    MANUAL = "manual"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
