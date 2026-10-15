from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.event_occurrence_item_labels import EventOccurrenceItemLabels
    from ..models.event_occurrence_item_metadata import EventOccurrenceItemMetadata
    from ..models.event_occurrence_item_payload import EventOccurrenceItemPayload


T = TypeVar("T", bound="EventOccurrenceItem")


@_attrs_define
class EventOccurrenceItem:
    """
    Attributes:
        consumer (str): Tenant consumer this occurrence was pushed into.
        created_at (datetime.datetime): When this occurrence was pushed, RFC3339.
        event (str): Event type name.
        event_id (str): Event occurrence id (UUID).
        event_version (int): Event type version this occurrence was accepted under.
        failed_deliveries (int): Failed delivery attempts for this occurrence. Counts attempts, not deliveries: a
            delivery that failed twice and then succeeded adds 2 here and 1 to successful_deliveries.
        payload (EventOccurrenceItemPayload): The pushed payload.
        pending_deliveries (int): Deliveries not finished yet: pending, sending or retrying.
        schema_valid (bool): Whether the payload validated against the event type's JSON Schema at push time.
        successful_deliveries (int): Successful delivery attempts for this occurrence.
        webhook_count (int): Number of distinct webhooks that got a delivery for this occurrence.
        schema (str | Unset): A URL to the JSON Schema for this object.
        labels (EventOccurrenceItemLabels | Unset): Key/value labels used to match against subscription label filters.
        metadata (EventOccurrenceItemMetadata | Unset): Arbitrary key/value metadata stored with the occurrence.
    """

    consumer: str
    created_at: datetime.datetime
    event: str
    event_id: str
    event_version: int
    failed_deliveries: int
    payload: EventOccurrenceItemPayload
    pending_deliveries: int
    schema_valid: bool
    successful_deliveries: int
    webhook_count: int
    schema: str | Unset = UNSET
    labels: EventOccurrenceItemLabels | Unset = UNSET
    metadata: EventOccurrenceItemMetadata | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        consumer = self.consumer

        created_at = self.created_at.isoformat()

        event = self.event

        event_id = self.event_id

        event_version = self.event_version

        failed_deliveries = self.failed_deliveries

        payload = self.payload.to_dict()

        pending_deliveries = self.pending_deliveries

        schema_valid = self.schema_valid

        successful_deliveries = self.successful_deliveries

        webhook_count = self.webhook_count

        schema = self.schema

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        metadata: dict[str, Any] | Unset = UNSET
        if not isinstance(self.metadata, Unset):
            metadata = self.metadata.to_dict()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "consumer": consumer,
                "created_at": created_at,
                "event": event,
                "event_id": event_id,
                "event_version": event_version,
                "failed_deliveries": failed_deliveries,
                "payload": payload,
                "pending_deliveries": pending_deliveries,
                "schema_valid": schema_valid,
                "successful_deliveries": successful_deliveries,
                "webhook_count": webhook_count,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if labels is not UNSET:
            field_dict["labels"] = labels
        if metadata is not UNSET:
            field_dict["metadata"] = metadata

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.event_occurrence_item_labels import EventOccurrenceItemLabels  # noqa: PLC0415
        from ..models.event_occurrence_item_metadata import EventOccurrenceItemMetadata  # noqa: PLC0415
        from ..models.event_occurrence_item_payload import EventOccurrenceItemPayload  # noqa: PLC0415

        d = dict(src_dict)
        consumer = d.pop("consumer")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))

        event = d.pop("event")

        event_id = d.pop("event_id")

        event_version = d.pop("event_version")

        failed_deliveries = d.pop("failed_deliveries")

        payload = EventOccurrenceItemPayload.from_dict(d.pop("payload"))

        pending_deliveries = d.pop("pending_deliveries")

        schema_valid = d.pop("schema_valid")

        successful_deliveries = d.pop("successful_deliveries")

        webhook_count = d.pop("webhook_count")

        schema = d.pop("$schema", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: EventOccurrenceItemLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = EventOccurrenceItemLabels.from_dict(_labels)

        _metadata = d.pop("metadata", UNSET)
        metadata: EventOccurrenceItemMetadata | Unset
        if isinstance(_metadata, Unset):
            metadata = UNSET
        else:
            metadata = EventOccurrenceItemMetadata.from_dict(_metadata)

        event_occurrence_item = cls(
            consumer=consumer,
            created_at=created_at,
            event=event,
            event_id=event_id,
            event_version=event_version,
            failed_deliveries=failed_deliveries,
            payload=payload,
            pending_deliveries=pending_deliveries,
            schema_valid=schema_valid,
            successful_deliveries=successful_deliveries,
            webhook_count=webhook_count,
            schema=schema,
            labels=labels,
            metadata=metadata,
        )

        return event_occurrence_item
