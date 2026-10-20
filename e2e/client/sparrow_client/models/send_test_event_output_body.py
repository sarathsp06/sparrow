from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="SendTestEventOutputBody")


@_attrs_define
class SendTestEventOutputBody:
    """
    Attributes:
        latency_ms (int): Time to the response headers, in milliseconds.
        response_excerpt (str): The first KiB of the response body.
        status_code (int): The receiver's HTTP status; 0 when no response was received.
        schema (str | Unset): A URL to the JSON Schema for this object.
        error (str | Unset): What went wrong when no response was received, or the template error.
        error_class (str | Unset): Delivery error category when the send did not get a 2xx: client_error, server_error,
            rate_limited, timeout, connection_refused, dns_error, tls_error, network_error, redirect, template_error or
            unknown.
    """

    latency_ms: int
    response_excerpt: str
    status_code: int
    schema: str | Unset = UNSET
    error: str | Unset = UNSET
    error_class: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        latency_ms = self.latency_ms

        response_excerpt = self.response_excerpt

        status_code = self.status_code

        schema = self.schema

        error = self.error

        error_class = self.error_class

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "latency_ms": latency_ms,
                "response_excerpt": response_excerpt,
                "status_code": status_code,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if error is not UNSET:
            field_dict["error"] = error
        if error_class is not UNSET:
            field_dict["error_class"] = error_class

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        latency_ms = d.pop("latency_ms")

        response_excerpt = d.pop("response_excerpt")

        status_code = d.pop("status_code")

        schema = d.pop("$schema", UNSET)

        error = d.pop("error", UNSET)

        error_class = d.pop("error_class", UNSET)

        send_test_event_output_body = cls(
            latency_ms=latency_ms,
            response_excerpt=response_excerpt,
            status_code=status_code,
            schema=schema,
            error=error,
            error_class=error_class,
        )

        return send_test_event_output_body
