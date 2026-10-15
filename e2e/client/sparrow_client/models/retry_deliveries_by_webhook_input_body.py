from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="RetryDeliveriesByWebhookInputBody")


@_attrs_define
class RetryDeliveriesByWebhookInputBody:
    """
    Attributes:
        webhook_id (str): Webhook id (UUID) whose deliveries to retry.
        schema (str | Unset): A URL to the JSON Schema for this object.
        force (bool | Unset): false (default) retries deliveries in status failed, pending, retrying or paused. true
            retries every delivery of the webhook, including successful and expired ones, which resends them.
    """

    webhook_id: str
    schema: str | Unset = UNSET
    force: bool | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        webhook_id = self.webhook_id

        schema = self.schema

        force = self.force

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "webhook_id": webhook_id,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if force is not UNSET:
            field_dict["force"] = force

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        webhook_id = d.pop("webhook_id")

        schema = d.pop("$schema", UNSET)

        force = d.pop("force", UNSET)

        retry_deliveries_by_webhook_input_body = cls(
            webhook_id=webhook_id,
            schema=schema,
            force=force,
        )

        return retry_deliveries_by_webhook_input_body
