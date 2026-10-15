from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.import_item_result import ImportItemResult
    from ..models.stamp_check_item import StampCheckItem


T = TypeVar("T", bound="ImportEventTypesOutputBody")


@_attrs_define
class ImportEventTypesOutputBody:
    """
    Attributes:
        applied (bool): True when the import was written. False for a dry run or when blocked_by is non-empty.
        dry_run (bool): Whether this was a dry run.
        items (list[ImportItemResult]): One result per bundle item, sorted by name.
        stamp (StampCheckItem):
        schema (str | Unset): A URL to the JSON Schema for this object.
        blocked_by (list[str] | Unset): Why nothing was written: unacknowledged stamp warnings, and breaking when a
            breaking change needs allow_breaking.
        imported_at (datetime.datetime | Unset): When the import was written, RFC3339. Use it to find deliveries that
            failed after the change.
    """

    applied: bool
    dry_run: bool
    items: list[ImportItemResult]
    stamp: StampCheckItem
    schema: str | Unset = UNSET
    blocked_by: list[str] | Unset = UNSET
    imported_at: datetime.datetime | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        applied = self.applied

        dry_run = self.dry_run

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        stamp = self.stamp.to_dict()

        schema = self.schema

        blocked_by: list[str] | Unset = UNSET
        if not isinstance(self.blocked_by, Unset):
            blocked_by = self.blocked_by

        imported_at: str | Unset = UNSET
        if not isinstance(self.imported_at, Unset):
            imported_at = self.imported_at.isoformat()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "applied": applied,
                "dry_run": dry_run,
                "items": items,
                "stamp": stamp,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if blocked_by is not UNSET:
            field_dict["blocked_by"] = blocked_by
        if imported_at is not UNSET:
            field_dict["imported_at"] = imported_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_item_result import ImportItemResult  # noqa: PLC0415
        from ..models.stamp_check_item import StampCheckItem  # noqa: PLC0415

        d = dict(src_dict)
        applied = d.pop("applied")

        dry_run = d.pop("dry_run")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = ImportItemResult.from_dict(items_item_data)

            items.append(items_item)

        stamp = StampCheckItem.from_dict(d.pop("stamp"))

        schema = d.pop("$schema", UNSET)

        blocked_by = cast(list[str], d.pop("blocked_by", UNSET))

        _imported_at = d.pop("imported_at", UNSET)
        imported_at: datetime.datetime | Unset
        if isinstance(_imported_at, Unset):
            imported_at = UNSET
        else:
            imported_at = datetime.datetime.fromisoformat(_imported_at)

        import_event_types_output_body = cls(
            applied=applied,
            dry_run=dry_run,
            items=items,
            stamp=stamp,
            schema=schema,
            blocked_by=blocked_by,
            imported_at=imported_at,
        )

        return import_event_types_output_body
