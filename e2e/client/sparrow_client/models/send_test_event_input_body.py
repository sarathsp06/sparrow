from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="SendTestEventInputBody")


@_attrs_define
class SendTestEventInputBody:
    """
    Attributes:
        subscription_id (str): Subscription whose event type sample is sent, rendered through its transform.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    subscription_id: str
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        subscription_id = self.subscription_id

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "subscription_id": subscription_id,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        subscription_id = d.pop("subscription_id")

        schema = d.pop("$schema", UNSET)

        send_test_event_input_body = cls(
            subscription_id=subscription_id,
            schema=schema,
        )

        return send_test_event_input_body
