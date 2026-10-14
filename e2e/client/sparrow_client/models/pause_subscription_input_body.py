from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="PauseSubscriptionInputBody")


@_attrs_define
class PauseSubscriptionInputBody:
    """
    Attributes:
        schema (str | Unset): A URL to the JSON Schema for this object.
        reason (str | Unset): Why the subscription is paused, shown next to it, for example: receiver maintenance until
            Friday.
    """

    schema: str | Unset = UNSET
    reason: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        schema = self.schema

        reason = self.reason

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        schema = d.pop("$schema", UNSET)

        reason = d.pop("reason", UNSET)

        pause_subscription_input_body = cls(
            schema=schema,
            reason=reason,
        )

        return pause_subscription_input_body
