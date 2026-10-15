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
        count (int): Number of deliveries retried.
        schema (str | Unset): A URL to the JSON Schema for this object.
        delivery_ids (list[str] | None | Unset): Ids of the deliveries that were retried.
    """

    count: int
    schema: str | Unset = UNSET
    delivery_ids: list[str] | None | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        count = self.count

        schema = self.schema

        delivery_ids: list[str] | None | Unset
        if isinstance(self.delivery_ids, Unset):
            delivery_ids = UNSET
        elif isinstance(self.delivery_ids, list):
            delivery_ids = self.delivery_ids

        else:
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

        def _parse_delivery_ids(data: object) -> list[str] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                delivery_ids_type_0 = cast(list[str], data)

                return delivery_ids_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None | Unset, data)

        delivery_ids = _parse_delivery_ids(d.pop("delivery_ids", UNSET))

        retry_output_body = cls(
            count=count,
            schema=schema,
            delivery_ids=delivery_ids,
        )

        return retry_output_body
