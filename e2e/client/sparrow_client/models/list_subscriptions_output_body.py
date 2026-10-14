from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.pagination_output import PaginationOutput
    from ..models.subscription_item import SubscriptionItem


T = TypeVar("T", bound="ListSubscriptionsOutputBody")


@_attrs_define
class ListSubscriptionsOutputBody:
    """
    Attributes:
        items (list[SubscriptionItem] | None):
        pagination (PaginationOutput):
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    items: list[SubscriptionItem] | None
    pagination: PaginationOutput
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

        pagination = self.pagination.to_dict()

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "items": items,
                "pagination": pagination,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.pagination_output import PaginationOutput  # noqa: PLC0415
        from ..models.subscription_item import SubscriptionItem  # noqa: PLC0415

        d = dict(src_dict)

        def _parse_items(data: object) -> list[SubscriptionItem] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                items_type_0 = []
                _items_type_0 = data
                for items_type_0_item_data in _items_type_0:
                    items_type_0_item = SubscriptionItem.from_dict(items_type_0_item_data)

                    items_type_0.append(items_type_0_item)

                return items_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[SubscriptionItem] | None, data)

        items = _parse_items(d.pop("items"))

        pagination = PaginationOutput.from_dict(d.pop("pagination"))

        schema = d.pop("$schema", UNSET)

        list_subscriptions_output_body = cls(
            items=items,
            pagination=pagination,
            schema=schema,
        )

        return list_subscriptions_output_body
