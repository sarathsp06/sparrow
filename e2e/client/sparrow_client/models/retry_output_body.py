from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="RetryOutputBody")


@_attrs_define
class RetryOutputBody:
    """
    Attributes:
        count (int): Number of deliveries queued for another round of attempts.
        schema (str | Unset): A URL to the JSON Schema for this object.
        delivery_ids (list[str] | Unset): Ids of the deliveries queued. Absent when nothing matched.
    """

    count: int
    schema: str | Unset = UNSET
    delivery_ids: list[str] | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        count = self.count

        schema = self.schema

        delivery_ids: list[str] | Unset = UNSET
        if not isinstance(self.delivery_ids, Unset):
            delivery_ids = self.delivery_ids

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "count": count,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if delivery_ids is not UNSET:
            field_dict["delivery_ids"] = delivery_ids

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        schema = d.pop("$schema", UNSET)

        delivery_ids = cast(list[str], d.pop("delivery_ids", UNSET))

        retry_output_body = cls(
            count=count,
            schema=schema,
            delivery_ids=delivery_ids,
        )

        return retry_output_body
