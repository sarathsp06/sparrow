from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="ConsumerStatsOutputBody")


@_attrs_define
class ConsumerStatsOutputBody:
    """
    Attributes:
        active_webhooks (int): Webhooks currently active (not paused).
        failed_deliveries (int): Attempts that failed, all time, including failures that were later retried
            successfully. Lags like total_deliveries.
        pending_deliveries (int): Deliveries waiting to be sent or retried right now (pending, sending or retrying).
        success_rate (float): successful_deliveries / total_deliveries, 0.0 to 1.0.
        successful_deliveries (int): Attempts that succeeded, all time. Lags like total_deliveries.
        total_deliveries (int): Delivery attempts recorded, all time. A retried delivery counts once per attempt.
            Updated by each health evaluation (SPARROW_HEALTH_EVAL_INTERVAL), so it can lag by that much.
        total_webhooks (int): Total webhooks registered.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    active_webhooks: int
    failed_deliveries: int
    pending_deliveries: int
    success_rate: float
    successful_deliveries: int
    total_deliveries: int
    total_webhooks: int
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        active_webhooks = self.active_webhooks

        failed_deliveries = self.failed_deliveries

        pending_deliveries = self.pending_deliveries

        success_rate = self.success_rate

        successful_deliveries = self.successful_deliveries

        total_deliveries = self.total_deliveries

        total_webhooks = self.total_webhooks

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "active_webhooks": active_webhooks,
                "failed_deliveries": failed_deliveries,
                "pending_deliveries": pending_deliveries,
                "success_rate": success_rate,
                "successful_deliveries": successful_deliveries,
                "total_deliveries": total_deliveries,
                "total_webhooks": total_webhooks,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        active_webhooks = d.pop("active_webhooks")

        failed_deliveries = d.pop("failed_deliveries")

        pending_deliveries = d.pop("pending_deliveries")

        success_rate = d.pop("success_rate")

        successful_deliveries = d.pop("successful_deliveries")

        total_deliveries = d.pop("total_deliveries")

        total_webhooks = d.pop("total_webhooks")

        schema = d.pop("$schema", UNSET)

        consumer_stats_output_body = cls(
            active_webhooks=active_webhooks,
            failed_deliveries=failed_deliveries,
            pending_deliveries=pending_deliveries,
            success_rate=success_rate,
            successful_deliveries=successful_deliveries,
            total_deliveries=total_deliveries,
            total_webhooks=total_webhooks,
            schema=schema,
        )

        return consumer_stats_output_body
