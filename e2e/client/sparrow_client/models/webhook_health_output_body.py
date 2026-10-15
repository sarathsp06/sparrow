from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..models.webhook_health_output_body_health import WebhookHealthOutputBodyHealth
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.health_rules import HealthRules


T = TypeVar("T", bound="WebhookHealthOutputBody")


@_attrs_define
class WebhookHealthOutputBody:
    """
    Attributes:
        avg_response_time (int): Average endpoint response time over the window, in milliseconds.
        client_errors (int): Failed attempts in the window with a 4xx response (client_error).
        consecutive_failures (int): Current run of consecutive failed attempts; resets on a success.
        failed_deliveries (int): Failed attempts in the window.
        health (WebhookHealthOutputBodyHealth): Health label from the last health evaluation: healthy, degraded,
            unhealthy, or unknown (too few recent attempts). Updated every SPARROW_HEALTH_EVAL_INTERVAL (default 1m).
        network_errors (int): Failed attempts in the window with a network-level error (DNS, TLS, connection refused,
            reset).
        rules (HealthRules):
        server_errors (int): Failed attempts in the window with a 5xx response (server_error).
        success_rate (float): successful_deliveries / total_deliveries over the window, 0.0 to 1.0.
        successful_deliveries (int): Successful attempts in the window.
        timeout_errors (int): Failed attempts in the window that timed out.
        total_deliveries (int): Delivery attempts in the last rules.window_hours hours (each retry counts).
        unexpected_status_errors (int): Attempts in the window whose 2xx/3xx response was not in expected_status_codes.
        webhook_id (str): Webhook id (UUID).
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    avg_response_time: int
    client_errors: int
    consecutive_failures: int
    failed_deliveries: int
    health: WebhookHealthOutputBodyHealth
    network_errors: int
    rules: HealthRules
    server_errors: int
    success_rate: float
    successful_deliveries: int
    timeout_errors: int
    total_deliveries: int
    unexpected_status_errors: int
    webhook_id: str
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        avg_response_time = self.avg_response_time

        client_errors = self.client_errors

        consecutive_failures = self.consecutive_failures

        failed_deliveries = self.failed_deliveries

        health = self.health.value

        network_errors = self.network_errors

        rules = self.rules.to_dict()

        server_errors = self.server_errors

        success_rate = self.success_rate

        successful_deliveries = self.successful_deliveries

        timeout_errors = self.timeout_errors

        total_deliveries = self.total_deliveries

        unexpected_status_errors = self.unexpected_status_errors

        webhook_id = self.webhook_id

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "avg_response_time": avg_response_time,
                "client_errors": client_errors,
                "consecutive_failures": consecutive_failures,
                "failed_deliveries": failed_deliveries,
                "health": health,
                "network_errors": network_errors,
                "rules": rules,
                "server_errors": server_errors,
                "success_rate": success_rate,
                "successful_deliveries": successful_deliveries,
                "timeout_errors": timeout_errors,
                "total_deliveries": total_deliveries,
                "unexpected_status_errors": unexpected_status_errors,
                "webhook_id": webhook_id,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.health_rules import HealthRules  # noqa: PLC0415

        d = dict(src_dict)
        avg_response_time = d.pop("avg_response_time")

        client_errors = d.pop("client_errors")

        consecutive_failures = d.pop("consecutive_failures")

        failed_deliveries = d.pop("failed_deliveries")

        health = WebhookHealthOutputBodyHealth(d.pop("health"))

        network_errors = d.pop("network_errors")

        rules = HealthRules.from_dict(d.pop("rules"))

        server_errors = d.pop("server_errors")

        success_rate = d.pop("success_rate")

        successful_deliveries = d.pop("successful_deliveries")

        timeout_errors = d.pop("timeout_errors")

        total_deliveries = d.pop("total_deliveries")

        unexpected_status_errors = d.pop("unexpected_status_errors")

        webhook_id = d.pop("webhook_id")

        schema = d.pop("$schema", UNSET)

        webhook_health_output_body = cls(
            avg_response_time=avg_response_time,
            client_errors=client_errors,
            consecutive_failures=consecutive_failures,
            failed_deliveries=failed_deliveries,
            health=health,
            network_errors=network_errors,
            rules=rules,
            server_errors=server_errors,
            success_rate=success_rate,
            successful_deliveries=successful_deliveries,
            timeout_errors=timeout_errors,
            total_deliveries=total_deliveries,
            unexpected_status_errors=unexpected_status_errors,
            webhook_id=webhook_id,
            schema=schema,
        )

        return webhook_health_output_body
