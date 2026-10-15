from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.alert_config_item import AlertConfigItem


T = TypeVar("T", bound="ListAlertConfigsOutputBody")


@_attrs_define
class ListAlertConfigsOutputBody:
    """
    Attributes:
        items (list[AlertConfigItem]): Alert configs, oldest first. Not paginated.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    items: list[AlertConfigItem]
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
        from ..models.alert_config_item import AlertConfigItem  # noqa: PLC0415

        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = AlertConfigItem.from_dict(items_item_data)

            items.append(items_item)

        schema = d.pop("$schema", UNSET)

        list_alert_configs_output_body = cls(
            items=items,
            schema=schema,
        )

        return list_alert_configs_output_body
