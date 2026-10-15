from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="AlertConfigItem")


@_attrs_define
class AlertConfigItem:
    """
    Attributes:
        consumer (str): Consumer this alert config belongs to.
        created_at (str): Creation timestamp, RFC3339.
        email (str): Recipient email address.
        event_types (list[str] | None): Sparrow system event types this recipient is alerted on.
        id (str): Alert config id (UUID).
        schema (str | Unset): A URL to the JSON Schema for this object.
        webhook_id (str | Unset): Webhook this alert is scoped to; omitted for consumer-wide alerts.
    """

    consumer: str
    created_at: str
    email: str
    event_types: list[str] | None
    id: str
    schema: str | Unset = UNSET
    webhook_id: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        consumer = self.consumer

        created_at = self.created_at

        email = self.email

        event_types: list[str] | None
        if isinstance(self.event_types, list):
            event_types = self.event_types

        else:
            event_types = self.event_types

        id = self.id

        schema = self.schema

        webhook_id = self.webhook_id

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "consumer": consumer,
                "created_at": created_at,
                "email": email,
                "event_types": event_types,
                "id": id,
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
        consumer = d.pop("consumer")

        created_at = d.pop("created_at")

        email = d.pop("email")

        def _parse_event_types(data: object) -> list[str] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                event_types_type_0 = cast(list[str], data)

                return event_types_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None, data)

        event_types = _parse_event_types(d.pop("event_types"))

        id = d.pop("id")

        schema = d.pop("$schema", UNSET)

        webhook_id = d.pop("webhook_id", UNSET)

        alert_config_item = cls(
            consumer=consumer,
            created_at=created_at,
            email=email,
            event_types=event_types,
            id=id,
            schema=schema,
            webhook_id=webhook_id,
        )

        return alert_config_item
