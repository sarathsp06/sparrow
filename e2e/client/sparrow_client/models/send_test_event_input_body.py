from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.send_test_event_input_body_payload import SendTestEventInputBodyPayload


T = TypeVar("T", bound="SendTestEventInputBody")


@_attrs_define
class SendTestEventInputBody:
    """
    Attributes:
        subscription_id (str): Subscription to send through: its transform renders the payload.
        schema (str | Unset): A URL to the JSON Schema for this object.
        event_name (str | Unset): Event to send. Required for a catch-all (*) subscription; otherwise it must be the
            subscription's event, if given.
        payload (SendTestEventInputBodyPayload | Unset): Event payload to send instead of the event type's sample
            payload. It is rendered through the subscription's transform like a pushed event.
    """

    subscription_id: str
    schema: str | Unset = UNSET
    event_name: str | Unset = UNSET
    payload: SendTestEventInputBodyPayload | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        subscription_id = self.subscription_id

        schema = self.schema

        event_name = self.event_name

        payload: dict[str, Any] | Unset = UNSET
        if not isinstance(self.payload, Unset):
            payload = self.payload.to_dict()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "subscription_id": subscription_id,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if event_name is not UNSET:
            field_dict["event_name"] = event_name
        if payload is not UNSET:
            field_dict["payload"] = payload

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.send_test_event_input_body_payload import SendTestEventInputBodyPayload  # noqa: PLC0415

        d = dict(src_dict)
        subscription_id = d.pop("subscription_id")

        schema = d.pop("$schema", UNSET)

        event_name = d.pop("event_name", UNSET)

        _payload = d.pop("payload", UNSET)
        payload: SendTestEventInputBodyPayload | Unset
        if isinstance(_payload, Unset):
            payload = UNSET
        else:
            payload = SendTestEventInputBodyPayload.from_dict(_payload)

        send_test_event_input_body = cls(
            subscription_id=subscription_id,
            schema=schema,
            event_name=event_name,
            payload=payload,
        )

        return send_test_event_input_body
