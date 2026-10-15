from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.cursor_pagination_output import CursorPaginationOutput
    from ..models.event_occurrence_item import EventOccurrenceItem


T = TypeVar("T", bound="ListEventOccurrencesOutputBody")


@_attrs_define
class ListEventOccurrencesOutputBody:
    """
    Attributes:
        items (list[EventOccurrenceItem]): Occurrences, newest first.
        pagination (CursorPaginationOutput):
        schema (str | Unset): A URL to the JSON Schema for this object.
        repush_id (str | Unset): Snapshot (batch job) id for POST .../events:rePush, present when prepare_repush was set
            and something matched. Expires after 15 minutes if not started.
        repush_total (int | Unset): Number of occurrences in that snapshot (all matches, not just this page).
    """

    items: list[EventOccurrenceItem]
    pagination: CursorPaginationOutput
    schema: str | Unset = UNSET
    repush_id: str | Unset = UNSET
    repush_total: int | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        pagination = self.pagination.to_dict()

        schema = self.schema

        repush_id = self.repush_id

        repush_total = self.repush_total

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "items": items,
                "pagination": pagination,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if repush_id is not UNSET:
            field_dict["repush_id"] = repush_id
        if repush_total is not UNSET:
            field_dict["repush_total"] = repush_total

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.cursor_pagination_output import CursorPaginationOutput  # noqa: PLC0415
        from ..models.event_occurrence_item import EventOccurrenceItem  # noqa: PLC0415

        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = EventOccurrenceItem.from_dict(items_item_data)

            items.append(items_item)

        pagination = CursorPaginationOutput.from_dict(d.pop("pagination"))

        schema = d.pop("$schema", UNSET)

        repush_id = d.pop("repush_id", UNSET)

        repush_total = d.pop("repush_total", UNSET)

        list_event_occurrences_output_body = cls(
            items=items,
            pagination=pagination,
            schema=schema,
            repush_id=repush_id,
            repush_total=repush_total,
        )

        return list_event_occurrences_output_body
