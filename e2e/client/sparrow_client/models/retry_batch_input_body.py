from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="RetryBatchInputBody")


@_attrs_define
class RetryBatchInputBody:
    """
    Attributes:
        retry_id (str): retry_id returned by an earlier GET .../deliveries?prepare_retry=true call.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    retry_id: str
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        retry_id = self.retry_id

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "retry_id": retry_id,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        retry_id = d.pop("retry_id")

        schema = d.pop("$schema", UNSET)

        retry_batch_input_body = cls(
            retry_id=retry_id,
            schema=schema,
        )

        return retry_batch_input_body
