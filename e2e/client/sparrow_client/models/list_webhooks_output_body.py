from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.pagination_output import PaginationOutput
    from ..models.webhook_out import WebhookOut


T = TypeVar("T", bound="ListWebhooksOutputBody")


@_attrs_define
class ListWebhooksOutputBody:
    """
    Attributes:
        items (list[WebhookOut]): Webhooks, newest first.
        pagination (PaginationOutput):
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    items: list[WebhookOut]
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
        from ..models.webhook_out import WebhookOut  # noqa: PLC0415

        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = WebhookOut.from_dict(items_item_data)

            items.append(items_item)

        pagination = PaginationOutput.from_dict(d.pop("pagination"))

        schema = d.pop("$schema", UNSET)

        list_webhooks_output_body = cls(
            items=items,
            pagination=pagination,
            schema=schema,
        )

        return list_webhooks_output_body
