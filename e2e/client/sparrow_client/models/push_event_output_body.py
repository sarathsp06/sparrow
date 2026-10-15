from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="PushEventOutputBody")


@_attrs_define
class PushEventOutputBody:
    """
    Attributes:
        duplicate (bool): True when idempotency_key matched an existing event occurrence in this consumer; no new event
            or deliveries were created and the status is still 201.
        event_id (str): Id (UUID) of the created event occurrence. When duplicate is true, this is the id of the
            PREVIOUSLY created event, not a new one.
        schema_valid (bool): Whether the payload validated against the event type's JSON Schema (true when it has none).
            False does not mean the push was rejected; see warnings. For a duplicate, the original event's result.
        schema (str | Unset): A URL to the JSON Schema for this object.
        warnings (list[str] | None | Unset): Non-fatal schema validation warnings. The event is stored and delivered
            regardless. Absent for a duplicate.
    """

    duplicate: bool
    event_id: str
    schema_valid: bool
    schema: str | Unset = UNSET
    warnings: list[str] | None | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        duplicate = self.duplicate

        event_id = self.event_id

        schema_valid = self.schema_valid

        schema = self.schema

        warnings: list[str] | None | Unset
        if isinstance(self.warnings, Unset):
            warnings = UNSET
        elif isinstance(self.warnings, list):
            warnings = self.warnings

        else:
            warnings = self.warnings

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "duplicate": duplicate,
                "event_id": event_id,
                "schema_valid": schema_valid,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if warnings is not UNSET:
            field_dict["warnings"] = warnings

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        duplicate = d.pop("duplicate")

        event_id = d.pop("event_id")

        schema_valid = d.pop("schema_valid")

        schema = d.pop("$schema", UNSET)

        def _parse_warnings(data: object) -> list[str] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                warnings_type_0 = cast(list[str], data)

                return warnings_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None | Unset, data)

        warnings = _parse_warnings(d.pop("warnings", UNSET))

        push_event_output_body = cls(
            duplicate=duplicate,
            event_id=event_id,
            schema_valid=schema_valid,
            schema=schema,
            warnings=warnings,
        )

        return push_event_output_body
