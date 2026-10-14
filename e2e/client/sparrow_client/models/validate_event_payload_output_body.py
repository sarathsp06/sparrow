from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="ValidateEventPayloadOutputBody")


@_attrs_define
class ValidateEventPayloadOutputBody:
    """
    Attributes:
        valid (bool): Whether the payload matches the event type's JSON Schema. Always true when the event type has no
            schema registered.
        schema (str | Unset): A URL to the JSON Schema for this object.
        warnings (list[str] | None | Unset): Per-field validation errors, present when valid is false.
    """

    valid: bool
    schema: str | Unset = UNSET
    warnings: list[str] | None | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        valid = self.valid

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
                "valid": valid,
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
        valid = d.pop("valid")

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

        validate_event_payload_output_body = cls(
            valid=valid,
            schema=schema,
            warnings=warnings,
        )

        return validate_event_payload_output_body
