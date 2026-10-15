from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

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
        items (list[EventOccurrenceItem] | None): Occurrences, newest first.
        pagination (CursorPaginationOutput):
        schema (str | Unset): A URL to the JSON Schema for this object.
        repush_id (str | Unset): Snapshot (batch job) id for POST .../events:rePush, present when prepare_repush was set
            and something matched. Expires after 15 minutes if not started.
        repush_total (int | Unset): Number of occurrences in that snapshot (all matches, not just this page).
    """

    items: list[EventOccurrenceItem] | None
    pagination: CursorPaginationOutput
    schema: str | Unset = UNSET
    repush_id: str | Unset = UNSET
    repush_total: int | Unset = UNSET

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

        def _parse_items(data: object) -> list[EventOccurrenceItem] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                items_type_0 = []
                _items_type_0 = data
                for items_type_0_item_data in _items_type_0:
                    items_type_0_item = EventOccurrenceItem.from_dict(items_type_0_item_data)

                    items_type_0.append(items_type_0_item)

                return items_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[EventOccurrenceItem] | None, data)

        items = _parse_items(d.pop("items"))

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
