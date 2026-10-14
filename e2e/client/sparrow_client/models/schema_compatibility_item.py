from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..models.schema_compatibility_item_result import SchemaCompatibilityItemResult
from ..types import UNSET, Unset

T = TypeVar("T", bound="SchemaCompatibilityItem")


@_attrs_define
class SchemaCompatibilityItem:
    """
    Attributes:
        result (SchemaCompatibilityItemResult): breaking when the change could break a payload transformation: a
            required property removed or made optional, a type widened, the schema removed, or a change under
            oneOf/anyOf/allOf/$ref/patternProperties that cannot be checked.
        reasons (list[str] | None | Unset): Each breaking change with the path it affects.
    """

    result: SchemaCompatibilityItemResult
    reasons: list[str] | None | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        result = self.result.value

        reasons: list[str] | None | Unset
        if isinstance(self.reasons, Unset):
            reasons = UNSET
        elif isinstance(self.reasons, list):
            reasons = self.reasons

        else:
            reasons = self.reasons

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "result": result,
            }
        )
        if reasons is not UNSET:
            field_dict["reasons"] = reasons

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        result = SchemaCompatibilityItemResult(d.pop("result"))

        def _parse_reasons(data: object) -> list[str] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                reasons_type_0 = cast(list[str], data)

                return reasons_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None | Unset, data)

        reasons = _parse_reasons(d.pop("reasons", UNSET))

        schema_compatibility_item = cls(
            result=result,
            reasons=reasons,
        )

        return schema_compatibility_item
