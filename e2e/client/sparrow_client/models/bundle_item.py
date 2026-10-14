from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.bundle_item_event_schema import BundleItemEventSchema
    from ..models.bundle_item_metadata import BundleItemMetadata


T = TypeVar("T", bound="BundleItem")


@_attrs_define
class BundleItem:
    """
    Attributes:
        name (str): Event type name.
        active (bool | Unset): Whether events of this type can be pushed. Omitted means true.
        description (str | Unset): Human-readable summary. Omitted means empty: an import replaces the definition.
        event_schema (BundleItemEventSchema | Unset): JSON Schema. Omitted means no schema, which removes an existing
            schema (a breaking change).
        metadata (BundleItemMetadata | Unset): Arbitrary key/value metadata. Omitted means none.
    """

    name: str
    active: bool | Unset = UNSET
    description: str | Unset = UNSET
    event_schema: BundleItemEventSchema | Unset = UNSET
    metadata: BundleItemMetadata | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        active = self.active

        description = self.description

        event_schema: dict[str, Any] | Unset = UNSET
        if not isinstance(self.event_schema, Unset):
            event_schema = self.event_schema.to_dict()

        metadata: dict[str, Any] | Unset = UNSET
        if not isinstance(self.metadata, Unset):
            metadata = self.metadata.to_dict()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "name": name,
            }
        )
        if active is not UNSET:
            field_dict["active"] = active
        if description is not UNSET:
            field_dict["description"] = description
        if event_schema is not UNSET:
            field_dict["event_schema"] = event_schema
        if metadata is not UNSET:
            field_dict["metadata"] = metadata

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.bundle_item_event_schema import BundleItemEventSchema  # noqa: PLC0415
        from ..models.bundle_item_metadata import BundleItemMetadata  # noqa: PLC0415

        d = dict(src_dict)
        name = d.pop("name")

        active = d.pop("active", UNSET)

        description = d.pop("description", UNSET)

        _event_schema = d.pop("event_schema", UNSET)
        event_schema: BundleItemEventSchema | Unset
        if isinstance(_event_schema, Unset):
            event_schema = UNSET
        else:
            event_schema = BundleItemEventSchema.from_dict(_event_schema)

        _metadata = d.pop("metadata", UNSET)
        metadata: BundleItemMetadata | Unset
        if isinstance(_metadata, Unset):
            metadata = UNSET
        else:
            metadata = BundleItemMetadata.from_dict(_metadata)

        bundle_item = cls(
            name=name,
            active=active,
            description=description,
            event_schema=event_schema,
            metadata=metadata,
        )

        return bundle_item
