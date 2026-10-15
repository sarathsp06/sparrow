from __future__ import annotations

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
        items (list[ImportItemResult] | None): One result per bundle item, sorted by name.
        stamp (StampCheckItem):
        schema (str | Unset): A URL to the JSON Schema for this object.
        blocked_by (list[str] | None | Unset): Why nothing was written: unacknowledged stamp warnings, and breaking when
            a breaking change needs allow_breaking.
        imported_at (str | Unset): When the import was written, RFC3339. Use it to find deliveries that failed after the
            change.
    """

    applied: bool
    dry_run: bool
    items: list[ImportItemResult] | None
    stamp: StampCheckItem
    schema: str | Unset = UNSET
    blocked_by: list[str] | None | Unset = UNSET
    imported_at: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        applied = self.applied

        dry_run = self.dry_run

        items: list[dict[str, Any]] | None
        if isinstance(self.items, list):
            items = []
            for items_type_0_item_data in self.items:
                items_type_0_item = items_type_0_item_data.to_dict()
                items.append(items_type_0_item)

        else:
            items = self.items

        stamp = self.stamp.to_dict()

        schema = self.schema

        blocked_by: list[str] | None | Unset
        if isinstance(self.blocked_by, Unset):
            blocked_by = UNSET
        elif isinstance(self.blocked_by, list):
            blocked_by = self.blocked_by

        else:
            blocked_by = self.blocked_by

        imported_at = self.imported_at

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

        def _parse_items(data: object) -> list[ImportItemResult] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                items_type_0 = []
                _items_type_0 = data
                for items_type_0_item_data in _items_type_0:
                    items_type_0_item = ImportItemResult.from_dict(items_type_0_item_data)

                    items_type_0.append(items_type_0_item)

                return items_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[ImportItemResult] | None, data)

        items = _parse_items(d.pop("items"))

        stamp = StampCheckItem.from_dict(d.pop("stamp"))

        schema = d.pop("$schema", UNSET)

        def _parse_blocked_by(data: object) -> list[str] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                blocked_by_type_0 = cast(list[str], data)

                return blocked_by_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None | Unset, data)

        blocked_by = _parse_blocked_by(d.pop("blocked_by", UNSET))

        imported_at = d.pop("imported_at", UNSET)

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
