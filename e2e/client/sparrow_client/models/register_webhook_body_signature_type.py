from enum import StrEnum


class RegisterWebhookBodySignatureType(StrEnum):
    ED25519 = "ed25519"
    HMAC = "hmac"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
