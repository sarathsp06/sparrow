from enum import StrEnum


class WebhookOutSignatureType(StrEnum):
    ED25519 = "ed25519"
    HMAC = "hmac"

    def __str__(self) -> str:
        return str(self.value)
