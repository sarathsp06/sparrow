from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.event_type_version_item import EventTypeVersionItem


T = TypeVar("T", bound="ListEventTypeVersionsOutputBody")


@_attrs_define
class ListEventTypeVersionsOutputBody:
    """
    Attributes:
        items (list[EventTypeVersionItem] | None): Every version of the event type, newest first.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    items: list[EventTypeVersionItem] | None
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        items: list[dict[str, Any]] | None
        if isinstance(self.items, list):
            items = []
            for items_type_0_item_data in self.items:
                items_type_0_item = items_type_0_item_data.to_dict()
                items.append(items_type_0_item)

        else:
            items = self.items

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

        def _parse_items(data: object) -> list[EventTypeVersionItem] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                items_type_0 = []
                _items_type_0 = data
                for items_type_0_item_data in _items_type_0:
                    items_type_0_item = EventTypeVersionItem.from_dict(items_type_0_item_data)

                    items_type_0.append(items_type_0_item)

                return items_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[EventTypeVersionItem] | None, data)

        items = _parse_items(d.pop("items"))

        schema = d.pop("$schema", UNSET)

        list_event_type_versions_output_body = cls(
            items=items,
            schema=schema,
        )

        return list_event_type_versions_output_body
