from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..models.event_type_bundle_api_version import EventTypeBundleApiVersion
from ..models.event_type_bundle_kind import EventTypeBundleKind
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.bundle_item import BundleItem
    from ..models.bundle_stamp_body import BundleStampBody


T = TypeVar("T", bound="EventTypeBundle")


@_attrs_define
class EventTypeBundle:
    """
    Attributes:
        api_version (EventTypeBundleApiVersion): Bundle schema version.
        items (list[BundleItem]): Event type definitions, sorted by name.
        kind (EventTypeBundleKind): Bundle kind.
        stamp (BundleStampBody):
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    api_version: EventTypeBundleApiVersion
    items: list[BundleItem]
    kind: EventTypeBundleKind
    stamp: BundleStampBody
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        api_version = self.api_version.value

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        kind = self.kind.value

        stamp = self.stamp.to_dict()

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "apiVersion": api_version,
                "items": items,
                "kind": kind,
                "stamp": stamp,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.bundle_item import BundleItem  # noqa: PLC0415
        from ..models.bundle_stamp_body import BundleStampBody  # noqa: PLC0415

        d = dict(src_dict)
        api_version = EventTypeBundleApiVersion(d.pop("apiVersion"))

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = BundleItem.from_dict(items_item_data)

            items.append(items_item)

        kind = EventTypeBundleKind(d.pop("kind"))

        stamp = BundleStampBody.from_dict(d.pop("stamp"))

        schema = d.pop("$schema", UNSET)

        event_type_bundle = cls(
            api_version=api_version,
            items=items,
            kind=kind,
            stamp=stamp,
            schema=schema,
        )

        return event_type_bundle
