from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.event_type_version_item import EventTypeVersionItem


T = TypeVar("T", bound="ListEventTypeVersionsOutputBody")


@_attrs_define
class ListEventTypeVersionsOutputBody:
    """
    Attributes:
        items (list[EventTypeVersionItem]): Every version of the event type, newest first.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    items: list[EventTypeVersionItem]
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
        from ..models.event_type_version_item import EventTypeVersionItem  # noqa: PLC0415

        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = EventTypeVersionItem.from_dict(items_item_data)

            items.append(items_item)

        schema = d.pop("$schema", UNSET)

        list_event_type_versions_output_body = cls(
            items=items,
            schema=schema,
        )

        return list_event_type_versions_output_body
