from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..models.batch_job_output_body_job_type import BatchJobOutputBodyJobType
from ..models.batch_job_output_body_status import BatchJobOutputBodyStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="BatchJobOutputBody")


@_attrs_define
class BatchJobOutputBody:
    """
    Attributes:
        consumer (str): Consumer the snapshot was taken in; empty for a snapshot taken from a cross-consumer list.
        created_at (str): Creation timestamp, RFC3339.
        expires_at (str): When the snapshot expires, RFC3339: 15 minutes after it was taken. An expired snapshot can no
            longer be started.
        failed (int): Items that could not be re-pushed or retried (e.g. the event or delivery no longer exists).
        id (str): Batch job id (UUID): the repush_id or retry_id from the list call.
        job_type (BatchJobOutputBodyJobType): Which batch operation this job performs.
        processed (int): Items handled successfully so far.
        status (BatchJobOutputBodyStatus): pending: snapshot taken, not started. processing: running. completed: every
            item was handled (see failed for the ones that could not be). failed: every item failed. cancelled: stopped by a
            cancel call.
        total (int): Total items in the batch snapshot.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    consumer: str
    created_at: str
    expires_at: str
    failed: int
    id: str
    job_type: BatchJobOutputBodyJobType
    processed: int
    status: BatchJobOutputBodyStatus
    total: int
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        consumer = self.consumer

        created_at = self.created_at

        expires_at = self.expires_at

        failed = self.failed

        id = self.id

        job_type = self.job_type.value

        processed = self.processed

        status = self.status.value

        total = self.total

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "consumer": consumer,
                "created_at": created_at,
                "expires_at": expires_at,
                "failed": failed,
                "id": id,
                "job_type": job_type,
                "processed": processed,
                "status": status,
                "total": total,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        consumer = d.pop("consumer")

        created_at = d.pop("created_at")

        expires_at = d.pop("expires_at")

        failed = d.pop("failed")

        id = d.pop("id")

        job_type = BatchJobOutputBodyJobType(d.pop("job_type"))

        processed = d.pop("processed")

        status = BatchJobOutputBodyStatus(d.pop("status"))

        total = d.pop("total")

        schema = d.pop("$schema", UNSET)

        batch_job_output_body = cls(
            consumer=consumer,
            created_at=created_at,
            expires_at=expires_at,
            failed=failed,
            id=id,
            job_type=job_type,
            processed=processed,
            status=status,
            total=total,
            schema=schema,
        )

        return batch_job_output_body
