from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="ResumeWebhookOutputBody")


@_attrs_define
class ResumeWebhookOutputBody:
    """
    Attributes:
        paused_deliveries (int): Deliveries held as paused for this webhook. They are not sent automatically; retry them
            to deliver.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    paused_deliveries: int
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        paused_deliveries = self.paused_deliveries

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "paused_deliveries": paused_deliveries,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        paused_deliveries = d.pop("paused_deliveries")

        schema = d.pop("$schema", UNSET)

        resume_webhook_output_body = cls(
            paused_deliveries=paused_deliveries,
            schema=schema,
        )

        return resume_webhook_output_body
