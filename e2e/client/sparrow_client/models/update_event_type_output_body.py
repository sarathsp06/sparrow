from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.event_type_change import EventTypeChange
    from ..models.update_event_type_output_body_event_schema import UpdateEventTypeOutputBodyEventSchema
    from ..models.update_event_type_output_body_metadata import UpdateEventTypeOutputBodyMetadata
    from ..models.update_event_type_output_body_sample_payload import UpdateEventTypeOutputBodySamplePayload


T = TypeVar("T", bound="UpdateEventTypeOutputBody")


@_attrs_define
class UpdateEventTypeOutputBody:
    """
    Attributes:
        active (bool): Whether events of this type can currently be pushed.
        change (EventTypeChange):
        created_at (datetime.datetime): Creation timestamp, RFC3339.
        name (str): Event type name, e.g. order.created.
        updated_at (datetime.datetime): Last-modified timestamp, RFC3339.
        version (int): Current schema version. Starts at 1 and increases only when the schema changes; every version is
            kept (see listEventTypeVersions).
        schema (str | Unset): A URL to the JSON Schema for this object.
        description (str | Unset): Human-readable summary of what this event represents.
        event_schema (UpdateEventTypeOutputBodyEventSchema | Unset): JSON Schema payloads are softly validated against,
            if one is registered.
        metadata (UpdateEventTypeOutputBodyMetadata | Unset): Arbitrary key/value metadata.
        sample_payload (UpdateEventTypeOutputBodySamplePayload | Unset): Example payload derived from the schema, used
            to preview subscription transform templates.
    """

    active: bool
    change: EventTypeChange
    created_at: datetime.datetime
    name: str
    updated_at: datetime.datetime
    version: int
    schema: str | Unset = UNSET
    description: str | Unset = UNSET
    event_schema: UpdateEventTypeOutputBodyEventSchema | Unset = UNSET
    metadata: UpdateEventTypeOutputBodyMetadata | Unset = UNSET
    sample_payload: UpdateEventTypeOutputBodySamplePayload | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        active = self.active

        change = self.change.to_dict()

        created_at = self.created_at.isoformat()

        name = self.name

        updated_at = self.updated_at.isoformat()

        version = self.version

        schema = self.schema

        description = self.description

        event_schema: dict[str, Any] | Unset = UNSET
        if not isinstance(self.event_schema, Unset):
            event_schema = self.event_schema.to_dict()

        metadata: dict[str, Any] | Unset = UNSET
        if not isinstance(self.metadata, Unset):
            metadata = self.metadata.to_dict()

        sample_payload: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sample_payload, Unset):
            sample_payload = self.sample_payload.to_dict()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "active": active,
                "change": change,
                "created_at": created_at,
                "name": name,
                "updated_at": updated_at,
                "version": version,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if description is not UNSET:
            field_dict["description"] = description
        if event_schema is not UNSET:
            field_dict["event_schema"] = event_schema
        if metadata is not UNSET:
            field_dict["metadata"] = metadata
        if sample_payload is not UNSET:
            field_dict["sample_payload"] = sample_payload

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.event_type_change import EventTypeChange  # noqa: PLC0415
        from ..models.update_event_type_output_body_event_schema import UpdateEventTypeOutputBodyEventSchema  # noqa: PLC0415
        from ..models.update_event_type_output_body_metadata import UpdateEventTypeOutputBodyMetadata  # noqa: PLC0415
        from ..models.update_event_type_output_body_sample_payload import UpdateEventTypeOutputBodySamplePayload  # noqa: PLC0415

        d = dict(src_dict)
        active = d.pop("active")

        change = EventTypeChange.from_dict(d.pop("change"))

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))

        name = d.pop("name")

        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))

        version = d.pop("version")

        schema = d.pop("$schema", UNSET)

        description = d.pop("description", UNSET)

        _event_schema = d.pop("event_schema", UNSET)
        event_schema: UpdateEventTypeOutputBodyEventSchema | Unset
        if isinstance(_event_schema, Unset):
            event_schema = UNSET
        else:
            event_schema = UpdateEventTypeOutputBodyEventSchema.from_dict(_event_schema)

        _metadata = d.pop("metadata", UNSET)
        metadata: UpdateEventTypeOutputBodyMetadata | Unset
        if isinstance(_metadata, Unset):
            metadata = UNSET
        else:
            metadata = UpdateEventTypeOutputBodyMetadata.from_dict(_metadata)

        _sample_payload = d.pop("sample_payload", UNSET)
        sample_payload: UpdateEventTypeOutputBodySamplePayload | Unset
        if isinstance(_sample_payload, Unset):
            sample_payload = UNSET
        else:
            sample_payload = UpdateEventTypeOutputBodySamplePayload.from_dict(_sample_payload)

        update_event_type_output_body = cls(
            active=active,
            change=change,
            created_at=created_at,
            name=name,
            updated_at=updated_at,
            version=version,
            schema=schema,
            description=description,
            event_schema=event_schema,
            metadata=metadata,
            sample_payload=sample_payload,
        )

        return update_event_type_output_body
