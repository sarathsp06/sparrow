from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.cursor_pagination_output import CursorPaginationOutput
    from ..models.delivery_item import DeliveryItem


T = TypeVar("T", bound="ListDeliveriesOutputBody")


@_attrs_define
class ListDeliveriesOutputBody:
    """
    Attributes:
        items (list[DeliveryItem] | None):
        pagination (CursorPaginationOutput):
        schema (str | Unset): A URL to the JSON Schema for this object.
        retry_id (str | Unset): Snapshot id for the batch retry endpoint, present when prepare_retry was set and
            something matched.
        retry_total (int | Unset): Number of deliveries in that snapshot.
    """

    items: list[DeliveryItem] | None
    pagination: CursorPaginationOutput
    schema: str | Unset = UNSET
    retry_id: str | Unset = UNSET
    retry_total: int | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        items: list[dict[str, Any]] | None
        if isinstance(self.items, list):
            items = []
            for items_type_0_item_data in self.items:
                items_type_0_item = items_type_0_item_data.to_dict()
                items.append(items_type_0_item)

        else:
            items = self.items

        pagination = self.pagination.to_dict()

        schema = self.schema

        retry_id = self.retry_id

        retry_total = self.retry_total

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "items": items,
                "pagination": pagination,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if retry_id is not UNSET:
            field_dict["retry_id"] = retry_id
        if retry_total is not UNSET:
            field_dict["retry_total"] = retry_total

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.cursor_pagination_output import CursorPaginationOutput  # noqa: PLC0415
        from ..models.delivery_item import DeliveryItem  # noqa: PLC0415

        d = dict(src_dict)

        def _parse_items(data: object) -> list[DeliveryItem] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                items_type_0 = []
                _items_type_0 = data
                for items_type_0_item_data in _items_type_0:
                    items_type_0_item = DeliveryItem.from_dict(items_type_0_item_data)

                    items_type_0.append(items_type_0_item)

                return items_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[DeliveryItem] | None, data)

        items = _parse_items(d.pop("items"))

        pagination = CursorPaginationOutput.from_dict(d.pop("pagination"))

        schema = d.pop("$schema", UNSET)

        retry_id = d.pop("retry_id", UNSET)

        retry_total = d.pop("retry_total", UNSET)

        list_deliveries_output_body = cls(
            items=items,
            pagination=pagination,
            schema=schema,
            retry_id=retry_id,
            retry_total=retry_total,
        )

        return list_deliveries_output_body
