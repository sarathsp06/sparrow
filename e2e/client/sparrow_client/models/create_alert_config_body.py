from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateAlertConfigBody")


@_attrs_define
class CreateAlertConfigBody:
    """
    Attributes:
        email (str): Recipient email address.
        event_types (list[str]): Sparrow system event types to alert on, at least one of:
            sparrow.webhook.health_changed, sparrow.webhook.delivery_failed, sparrow.webhook.disabled.
        schema (str | Unset): A URL to the JSON Schema for this object.
        webhook_id (str | Unset): Scope the alert to one webhook. Omit for a consumer-wide alert covering every webhook.
    """

    email: str
    event_types: list[str]
    schema: str | Unset = UNSET
    webhook_id: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        email = self.email

        event_types = self.event_types

        schema = self.schema

        webhook_id = self.webhook_id

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "email": email,
                "event_types": event_types,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if webhook_id is not UNSET:
            field_dict["webhook_id"] = webhook_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        event_types = cast(list[str], d.pop("event_types"))

        schema = d.pop("$schema", UNSET)

        webhook_id = d.pop("webhook_id", UNSET)

        create_alert_config_body = cls(
            email=email,
            event_types=event_types,
            schema=schema,
            webhook_id=webhook_id,
        )

        return create_alert_config_body
