from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="HealthRules")


@_attrs_define
class HealthRules:
    """
    Attributes:
        degraded_min_attempts (int): Attempts in the window needed before a low success rate counts as degraded.
        degraded_success_rate (float): A success rate below this (0.0 to 1.0), over at least degraded_min_attempts
            attempts, is degraded.
        healthy_min_attempts (int): Attempts at or above degraded_success_rate needed to call a webhook healthy; with
            fewer it stays unknown.
        unhealthy_consecutive_failures (int): Failed attempts in a row that make a webhook unhealthy whatever its
            success rate.
        unhealthy_min_attempts (int): Attempts in the window needed before a low success rate counts as unhealthy.
        unhealthy_success_rate (float): A success rate below this (0.0 to 1.0), over at least unhealthy_min_attempts
            attempts, is unhealthy.
        window_hours (int): Lookback window, in hours, of the attempt counts and success rate.
    """

    degraded_min_attempts: int
    degraded_success_rate: float
    healthy_min_attempts: int
    unhealthy_consecutive_failures: int
    unhealthy_min_attempts: int
    unhealthy_success_rate: float
    window_hours: int

    def to_dict(self) -> dict[str, Any]:
        degraded_min_attempts = self.degraded_min_attempts

        degraded_success_rate = self.degraded_success_rate

        healthy_min_attempts = self.healthy_min_attempts

        unhealthy_consecutive_failures = self.unhealthy_consecutive_failures

        unhealthy_min_attempts = self.unhealthy_min_attempts

        unhealthy_success_rate = self.unhealthy_success_rate

        window_hours = self.window_hours

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "degraded_min_attempts": degraded_min_attempts,
                "degraded_success_rate": degraded_success_rate,
                "healthy_min_attempts": healthy_min_attempts,
                "unhealthy_consecutive_failures": unhealthy_consecutive_failures,
                "unhealthy_min_attempts": unhealthy_min_attempts,
                "unhealthy_success_rate": unhealthy_success_rate,
                "window_hours": window_hours,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        degraded_min_attempts = d.pop("degraded_min_attempts")

        degraded_success_rate = d.pop("degraded_success_rate")

        healthy_min_attempts = d.pop("healthy_min_attempts")

        unhealthy_consecutive_failures = d.pop("unhealthy_consecutive_failures")

        unhealthy_min_attempts = d.pop("unhealthy_min_attempts")

        unhealthy_success_rate = d.pop("unhealthy_success_rate")

        window_hours = d.pop("window_hours")

        health_rules = cls(
            degraded_min_attempts=degraded_min_attempts,
            degraded_success_rate=degraded_success_rate,
            healthy_min_attempts=healthy_min_attempts,
            unhealthy_consecutive_failures=unhealthy_consecutive_failures,
            unhealthy_min_attempts=unhealthy_min_attempts,
            unhealthy_success_rate=unhealthy_success_rate,
            window_hours=window_hours,
        )

        return health_rules
