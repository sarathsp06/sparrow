from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.validate_event_payload_input_body_payload import ValidateEventPayloadInputBodyPayload


T = TypeVar("T", bound="ValidateEventPayloadInputBody")


@_attrs_define
class ValidateEventPayloadInputBody:
    """
    Attributes:
        payload (ValidateEventPayloadInputBodyPayload): Payload to validate against the event type's registered JSON
            Schema.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    payload: ValidateEventPayloadInputBodyPayload
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        payload = self.payload.to_dict()

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "payload": payload,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.validate_event_payload_input_body_payload import ValidateEventPayloadInputBodyPayload  # noqa: PLC0415

        d = dict(src_dict)
        payload = ValidateEventPayloadInputBodyPayload.from_dict(d.pop("payload"))

        schema = d.pop("$schema", UNSET)

        validate_event_payload_input_body = cls(
            payload=payload,
            schema=schema,
        )

        return validate_event_payload_input_body
