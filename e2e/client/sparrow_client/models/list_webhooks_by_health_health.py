from enum import StrEnum


class ListWebhooksByHealthHealth(StrEnum):
    DEGRADED = "degraded"
    HEALTHY = "healthy"
    UNHEALTHY = "unhealthy"
    UNKNOWN = "unknown"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
