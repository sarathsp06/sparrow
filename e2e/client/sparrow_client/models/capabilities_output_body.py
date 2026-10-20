from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ai_drafting_struct import AIDraftingStruct
    from ..models.alert_delivery_struct import AlertDeliveryStruct
    from ..models.receiver_code_struct import ReceiverCodeStruct


T = TypeVar("T", bound="CapabilitiesOutputBody")


@_attrs_define
class CapabilitiesOutputBody:
    """
    Attributes:
        ai_drafting (AIDraftingStruct):
        alert_delivery (AlertDeliveryStruct):
        receiver_code (ReceiverCodeStruct):
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    ai_drafting: AIDraftingStruct
    alert_delivery: AlertDeliveryStruct
    receiver_code: ReceiverCodeStruct
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        ai_drafting = self.ai_drafting.to_dict()

        alert_delivery = self.alert_delivery.to_dict()

        receiver_code = self.receiver_code.to_dict()

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "ai_drafting": ai_drafting,
                "alert_delivery": alert_delivery,
                "receiver_code": receiver_code,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_drafting_struct import AIDraftingStruct  # noqa: PLC0415
        from ..models.alert_delivery_struct import AlertDeliveryStruct  # noqa: PLC0415
        from ..models.receiver_code_struct import ReceiverCodeStruct  # noqa: PLC0415

        d = dict(src_dict)
        ai_drafting = AIDraftingStruct.from_dict(d.pop("ai_drafting"))

        alert_delivery = AlertDeliveryStruct.from_dict(d.pop("alert_delivery"))

        receiver_code = ReceiverCodeStruct.from_dict(d.pop("receiver_code"))

        schema = d.pop("$schema", UNSET)

        capabilities_output_body = cls(
            ai_drafting=ai_drafting,
            alert_delivery=alert_delivery,
            receiver_code=receiver_code,
            schema=schema,
        )

        return capabilities_output_body
