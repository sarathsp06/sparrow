from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ai_drafting_struct import AIDraftingStruct
    from ..models.alert_delivery_struct import AlertDeliveryStruct


T = TypeVar("T", bound="CapabilitiesOutputBody")


@_attrs_define
class CapabilitiesOutputBody:
    """
    Attributes:
        ai_drafting (AIDraftingStruct):
        alert_delivery (AlertDeliveryStruct):
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    ai_drafting: AIDraftingStruct
    alert_delivery: AlertDeliveryStruct
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        ai_drafting = self.ai_drafting.to_dict()

        alert_delivery = self.alert_delivery.to_dict()

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "ai_drafting": ai_drafting,
                "alert_delivery": alert_delivery,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_drafting_struct import AIDraftingStruct  # noqa: PLC0415
        from ..models.alert_delivery_struct import AlertDeliveryStruct  # noqa: PLC0415

        d = dict(src_dict)
        ai_drafting = AIDraftingStruct.from_dict(d.pop("ai_drafting"))

        alert_delivery = AlertDeliveryStruct.from_dict(d.pop("alert_delivery"))

        schema = d.pop("$schema", UNSET)

        capabilities_output_body = cls(
            ai_drafting=ai_drafting,
            alert_delivery=alert_delivery,
            schema=schema,
        )

        return capabilities_output_body
