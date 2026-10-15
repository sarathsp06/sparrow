from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

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
        items (list[SubscriptionItem]): Subscriptions, newest first.
        pagination (PaginationOutput):
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    items: list[SubscriptionItem]
    pagination: PaginationOutput
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

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
        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = SubscriptionItem.from_dict(items_item_data)

            items.append(items_item)

        pagination = PaginationOutput.from_dict(d.pop("pagination"))

        schema = d.pop("$schema", UNSET)

        list_subscriptions_output_body = cls(
            items=items,
            pagination=pagination,
            schema=schema,
        )

        return list_subscriptions_output_body
