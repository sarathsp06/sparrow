from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="RegisterWebhookBodyHeaders")


@_attrs_define
class RegisterWebhookBodyHeaders:
    """Static HTTP headers sent with every delivery to this webhook. Only string values are sent. Names must be valid HTTP
    tokens, values at most 8 KiB with no CR/LF; framing headers (Host, Content-Length, Transfer-Encoding, Connection,
    Keep-Alive, Proxy-Connection, TE, Trailer, Upgrade) are rejected. Stored and returned in plaintext: put credentials
    in secret_headers.

    """

    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        register_webhook_body_headers = cls()

        register_webhook_body_headers.additional_properties = d
        return register_webhook_body_headers

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
