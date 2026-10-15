from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.push_event_body_labels import PushEventBodyLabels
    from ..models.push_event_body_metadata import PushEventBodyMetadata
    from ..models.push_event_body_payload import PushEventBodyPayload


T = TypeVar("T", bound="PushEventBody")


@_attrs_define
class PushEventBody:
    """
    Attributes:
        payload (PushEventBodyPayload): Event payload, a JSON object. Validated softly against the event type's JSON
            Schema if one is registered. The whole request body is limited by SPARROW_MAX_BODY_BYTES (default 5 MiB).
        schema (str | Unset): A URL to the JSON Schema for this object.
        idempotency_key (str | Unset): Client-supplied deduplication key, unique per consumer. A push whose key matches
            an earlier event in the same consumer still returns 201, with duplicate: true and the original event_id, and
            creates no event or deliveries. The key is kept as long as the event (see SPARROW_EVENT_RETENTION_DAYS).
        labels (PushEventBodyLabels | Unset): Key/value labels a subscription's label_filters match against to decide
            which subscriptions receive this occurrence. At most 20 pairs; keys up to 64 characters of letters, digits, '.',
            '_' and '-'; values up to 256 characters.
        metadata (PushEventBodyMetadata | Unset): Arbitrary key/value metadata stored with the occurrence.
        ttl_seconds (int | Unset): Time-to-live in seconds, counted from fan-out. A delivery attempt that would start
            after it expires is not made and the delivery fails as expired. 0 or omitted means no expiry. Manual retries
            ignore it.
    """

    payload: PushEventBodyPayload
    schema: str | Unset = UNSET
    idempotency_key: str | Unset = UNSET
    labels: PushEventBodyLabels | Unset = UNSET
    metadata: PushEventBodyMetadata | Unset = UNSET
    ttl_seconds: int | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        payload = self.payload.to_dict()

        schema = self.schema

        idempotency_key = self.idempotency_key

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        metadata: dict[str, Any] | Unset = UNSET
        if not isinstance(self.metadata, Unset):
            metadata = self.metadata.to_dict()

        ttl_seconds = self.ttl_seconds

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "payload": payload,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if idempotency_key is not UNSET:
            field_dict["idempotency_key"] = idempotency_key
        if labels is not UNSET:
            field_dict["labels"] = labels
        if metadata is not UNSET:
            field_dict["metadata"] = metadata
        if ttl_seconds is not UNSET:
            field_dict["ttl_seconds"] = ttl_seconds

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_event_body_labels import PushEventBodyLabels  # noqa: PLC0415
        from ..models.push_event_body_metadata import PushEventBodyMetadata  # noqa: PLC0415
        from ..models.push_event_body_payload import PushEventBodyPayload  # noqa: PLC0415

        d = dict(src_dict)
        payload = PushEventBodyPayload.from_dict(d.pop("payload"))

        schema = d.pop("$schema", UNSET)

        idempotency_key = d.pop("idempotency_key", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: PushEventBodyLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = PushEventBodyLabels.from_dict(_labels)

        _metadata = d.pop("metadata", UNSET)
        metadata: PushEventBodyMetadata | Unset
        if isinstance(_metadata, Unset):
            metadata = UNSET
        else:
            metadata = PushEventBodyMetadata.from_dict(_metadata)

        ttl_seconds = d.pop("ttl_seconds", UNSET)

        push_event_body = cls(
            payload=payload,
            schema=schema,
            idempotency_key=idempotency_key,
            labels=labels,
            metadata=metadata,
            ttl_seconds=ttl_seconds,
        )

        return push_event_body
