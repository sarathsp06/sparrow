from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.health_rules import HealthRules


T = TypeVar("T", bound="HealthSummaryOutputBody")


@_attrs_define
class HealthSummaryOutputBody:
    """
    Attributes:
        degraded_count (int): Webhooks currently labelled degraded.
        healthy_count (int): Webhooks currently labelled healthy.
        rules (HealthRules):
        unhealthy_count (int): Webhooks currently labelled unhealthy.
        unknown_count (int): Webhooks labelled unknown: too few recent attempts to judge.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    degraded_count: int
    healthy_count: int
    rules: HealthRules
    unhealthy_count: int
    unknown_count: int
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        degraded_count = self.degraded_count

        healthy_count = self.healthy_count

        rules = self.rules.to_dict()

        unhealthy_count = self.unhealthy_count

        unknown_count = self.unknown_count

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "degraded_count": degraded_count,
                "healthy_count": healthy_count,
                "rules": rules,
                "unhealthy_count": unhealthy_count,
                "unknown_count": unknown_count,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.health_rules import HealthRules  # noqa: PLC0415

        d = dict(src_dict)
        degraded_count = d.pop("degraded_count")

        healthy_count = d.pop("healthy_count")

        rules = HealthRules.from_dict(d.pop("rules"))

        unhealthy_count = d.pop("unhealthy_count")

        unknown_count = d.pop("unknown_count")

        schema = d.pop("$schema", UNSET)

        health_summary_output_body = cls(
            degraded_count=degraded_count,
            healthy_count=healthy_count,
            rules=rules,
            unhealthy_count=unhealthy_count,
            unknown_count=unknown_count,
            schema=schema,
        )

        return health_summary_output_body
