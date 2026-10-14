from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="AlertDeliveryStruct")


@_attrs_define
class AlertDeliveryStruct:
    """
    Attributes:
        configured (bool): True when an active webhook under the _sparrow consumer has an unpaused subscription to
            Sparrow's system events, so alert configs actually send email. When false, alert configs are stored but nothing
            is sent.
    """

    configured: bool

    def to_dict(self) -> dict[str, Any]:
        configured = self.configured

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "configured": configured,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        configured = d.pop("configured")

        alert_delivery_struct = cls(
            configured=configured,
        )

        return alert_delivery_struct
