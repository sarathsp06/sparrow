from enum import StrEnum


class AttemptItemErrorCategory(StrEnum):
    CLIENT_ERROR = "client_error"
    CONNECTION_REFUSED = "connection_refused"
    DNS_ERROR = "dns_error"
    NETWORK_ERROR = "network_error"
    RATE_LIMITED = "rate_limited"
    SERVER_ERROR = "server_error"
    SUCCESS = "success"
    TEMPLATE_ERROR = "template_error"
    TIMEOUT = "timeout"
    TLS_ERROR = "tls_error"
    UNEXPECTED_STATUS = "unexpected_status"
    UNKNOWN = "unknown"

    def __str__(self) -> str:
        return str(self.value)
