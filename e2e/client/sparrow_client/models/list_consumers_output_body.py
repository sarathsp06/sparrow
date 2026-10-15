from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.consumer_item import ConsumerItem


T = TypeVar("T", bound="ListConsumersOutputBody")


@_attrs_define
class ListConsumersOutputBody:
    """
    Attributes:
        items (list[ConsumerItem]): Matching consumer names, sorted by name.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    items: list[ConsumerItem]
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "items": items,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.consumer_item import ConsumerItem  # noqa: PLC0415

        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = ConsumerItem.from_dict(items_item_data)

            items.append(items_item)

        schema = d.pop("$schema", UNSET)

        list_consumers_output_body = cls(
            items=items,
            schema=schema,
        )

        return list_consumers_output_body
