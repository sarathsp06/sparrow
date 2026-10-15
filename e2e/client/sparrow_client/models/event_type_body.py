from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.event_type_body_event_schema import EventTypeBodyEventSchema
    from ..models.event_type_body_metadata import EventTypeBodyMetadata


T = TypeVar("T", bound="EventTypeBody")


@_attrs_define
class EventTypeBody:
    """
    Attributes:
        schema (str | Unset): A URL to the JSON Schema for this object.
        active (bool | Unset): Whether events of this type can be pushed; a push to an inactive type fails with 409.
            Defaults to true on register; unchanged on update when omitted.
        description (str | Unset): Human-readable summary of what this event represents. On update, an empty string
            leaves it unchanged.
        event_schema (EventTypeBodyEventSchema | Unset): Optional JSON Schema for the payload. Validation is soft: non-
            conforming payloads are still accepted, with warnings returned from the push call. On update, sending a
            different schema creates a new version.
        metadata (EventTypeBodyMetadata | Unset): Arbitrary key/value metadata for your own tooling. On update, replaces
            the whole map.
        name (str | Unset): Unique event type name, e.g. order.created. Required on register, ignored on update (the
            path names the type). At most 255 characters; the sparrow. prefix is reserved. Immutable after creation.
    """

    schema: str | Unset = UNSET
    active: bool | Unset = UNSET
    description: str | Unset = UNSET
    event_schema: EventTypeBodyEventSchema | Unset = UNSET
    metadata: EventTypeBodyMetadata | Unset = UNSET
    name: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        schema = self.schema

        active = self.active

        description = self.description

        event_schema: dict[str, Any] | Unset = UNSET
        if not isinstance(self.event_schema, Unset):
            event_schema = self.event_schema.to_dict()

        metadata: dict[str, Any] | Unset = UNSET
        if not isinstance(self.metadata, Unset):
            metadata = self.metadata.to_dict()

        name = self.name

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if active is not UNSET:
            field_dict["active"] = active
        if description is not UNSET:
            field_dict["description"] = description
        if event_schema is not UNSET:
            field_dict["event_schema"] = event_schema
        if metadata is not UNSET:
            field_dict["metadata"] = metadata
        if name is not UNSET:
            field_dict["name"] = name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.event_type_body_event_schema import EventTypeBodyEventSchema  # noqa: PLC0415
        from ..models.event_type_body_metadata import EventTypeBodyMetadata  # noqa: PLC0415

        d = dict(src_dict)
        schema = d.pop("$schema", UNSET)

        active = d.pop("active", UNSET)

        description = d.pop("description", UNSET)

        _event_schema = d.pop("event_schema", UNSET)
        event_schema: EventTypeBodyEventSchema | Unset
        if isinstance(_event_schema, Unset):
            event_schema = UNSET
        else:
            event_schema = EventTypeBodyEventSchema.from_dict(_event_schema)

        _metadata = d.pop("metadata", UNSET)
        metadata: EventTypeBodyMetadata | Unset
        if isinstance(_metadata, Unset):
            metadata = UNSET
        else:
            metadata = EventTypeBodyMetadata.from_dict(_metadata)

        name = d.pop("name", UNSET)

        event_type_body = cls(
            schema=schema,
            active=active,
            description=description,
            event_schema=event_schema,
            metadata=metadata,
            name=name,
        )

        return event_type_body
