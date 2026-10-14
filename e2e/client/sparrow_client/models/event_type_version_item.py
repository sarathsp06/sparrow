from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.event_type_version_item_event_schema import EventTypeVersionItemEventSchema
    from ..models.event_type_version_item_sample_payload import EventTypeVersionItemSamplePayload


T = TypeVar("T", bound="EventTypeVersionItem")


@_attrs_define
class EventTypeVersionItem:
    """
    Attributes:
        created_at (str): When this version was created, RFC3339.
        name (str): Event type name.
        version (int): Version number.
        schema (str | Unset): A URL to the JSON Schema for this object.
        description (str | Unset): Description as it was when this version was created.
        event_schema (EventTypeVersionItemEventSchema | Unset): JSON Schema of this version. Absent for a schema-less
            version.
        sample_payload (EventTypeVersionItemSamplePayload | Unset): Example payload generated from this version's
            schema.
        schema_defined_at (str | Unset): When a first schema was added to this version in place, RFC3339. Events pushed
            before this time were accepted while the version had no schema.
    """

    created_at: str
    name: str
    version: int
    schema: str | Unset = UNSET
    description: str | Unset = UNSET
    event_schema: EventTypeVersionItemEventSchema | Unset = UNSET
    sample_payload: EventTypeVersionItemSamplePayload | Unset = UNSET
    schema_defined_at: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at

        name = self.name

        version = self.version

        schema = self.schema

        description = self.description

        event_schema: dict[str, Any] | Unset = UNSET
        if not isinstance(self.event_schema, Unset):
            event_schema = self.event_schema.to_dict()

        sample_payload: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sample_payload, Unset):
            sample_payload = self.sample_payload.to_dict()

        schema_defined_at = self.schema_defined_at

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "created_at": created_at,
                "name": name,
                "version": version,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if description is not UNSET:
            field_dict["description"] = description
        if event_schema is not UNSET:
            field_dict["event_schema"] = event_schema
        if sample_payload is not UNSET:
            field_dict["sample_payload"] = sample_payload
        if schema_defined_at is not UNSET:
            field_dict["schema_defined_at"] = schema_defined_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.event_type_version_item_event_schema import EventTypeVersionItemEventSchema  # noqa: PLC0415
        from ..models.event_type_version_item_sample_payload import EventTypeVersionItemSamplePayload  # noqa: PLC0415

        d = dict(src_dict)
        created_at = d.pop("created_at")

        name = d.pop("name")

        version = d.pop("version")

        schema = d.pop("$schema", UNSET)

        description = d.pop("description", UNSET)

        _event_schema = d.pop("event_schema", UNSET)
        event_schema: EventTypeVersionItemEventSchema | Unset
        if isinstance(_event_schema, Unset):
            event_schema = UNSET
        else:
            event_schema = EventTypeVersionItemEventSchema.from_dict(_event_schema)

        _sample_payload = d.pop("sample_payload", UNSET)
        sample_payload: EventTypeVersionItemSamplePayload | Unset
        if isinstance(_sample_payload, Unset):
            sample_payload = UNSET
        else:
            sample_payload = EventTypeVersionItemSamplePayload.from_dict(_sample_payload)

        schema_defined_at = d.pop("schema_defined_at", UNSET)

        event_type_version_item = cls(
            created_at=created_at,
            name=name,
            version=version,
            schema=schema,
            description=description,
            event_schema=event_schema,
            sample_payload=sample_payload,
            schema_defined_at=schema_defined_at,
        )

        return event_type_version_item
