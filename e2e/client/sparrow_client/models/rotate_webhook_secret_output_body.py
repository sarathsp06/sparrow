from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="RotateWebhookSecretOutputBody")


@_attrs_define
class RotateWebhookSecretOutputBody:
    """
    Attributes:
        webhook_secret (str): The new HMAC signing secret, in plaintext. Shown only in this response.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    webhook_secret: str
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        webhook_secret = self.webhook_secret

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "webhook_secret": webhook_secret,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        webhook_secret = d.pop("webhook_secret")

        schema = d.pop("$schema", UNSET)

        rotate_webhook_secret_output_body = cls(
            webhook_secret=webhook_secret,
            schema=schema,
        )

        return rotate_webhook_secret_output_body
