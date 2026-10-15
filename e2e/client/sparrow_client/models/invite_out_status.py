from enum import StrEnum


class InviteOutStatus(StrEnum):
    CANCELLED = "cancelled"
    EXPIRED = "expired"
    PENDING = "pending"
    REDEEMED = "redeemed"

    def __str__(self) -> str:
        return str(self.value)
