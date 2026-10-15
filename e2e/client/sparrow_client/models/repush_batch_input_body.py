from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="RepushBatchInputBody")


@_attrs_define
class RepushBatchInputBody:
    """
    Attributes:
        repush_id (str): repush_id returned by an earlier GET .../events?prepare_repush=true call.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    repush_id: str
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        repush_id = self.repush_id

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "repush_id": repush_id,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        repush_id = d.pop("repush_id")

        schema = d.pop("$schema", UNSET)

        repush_batch_input_body = cls(
            repush_id=repush_id,
            schema=schema,
        )

        return repush_batch_input_body
