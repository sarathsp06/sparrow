from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.template_failure_item import TemplateFailureItem


T = TypeVar("T", bound="TemplateCheckItem")


@_attrs_define
class TemplateCheckItem:
    """
    Attributes:
        catch_all (int): How many of the affected subscriptions are catch-all ("*").
        passed (int): Subscriptions whose transform renders.
        without_transform (int): Subscriptions with no transform. Sparrow passes the payload through and cannot tell
            whether the receiver copes.
        failures (list[TemplateFailureItem] | Unset): Subscriptions whose transform no longer renders against the new
            version.
        paused (list[str] | Unset): Subscriptions this import paused (subscription_policy=pause).
    """

    catch_all: int
    passed: int
    without_transform: int
    failures: list[TemplateFailureItem] | Unset = UNSET
    paused: list[str] | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        catch_all = self.catch_all

        passed = self.passed

        without_transform = self.without_transform

        failures: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.failures, Unset):
            failures = []
            for failures_item_data in self.failures:
                failures_item = failures_item_data.to_dict()
                failures.append(failures_item)

        paused: list[str] | Unset = UNSET
        if not isinstance(self.paused, Unset):
            paused = self.paused

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "catch_all": catch_all,
                "passed": passed,
                "without_transform": without_transform,
            }
        )
        if failures is not UNSET:
            field_dict["failures"] = failures
        if paused is not UNSET:
            field_dict["paused"] = paused

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.template_failure_item import TemplateFailureItem  # noqa: PLC0415

        d = dict(src_dict)
        catch_all = d.pop("catch_all")

        passed = d.pop("passed")

        without_transform = d.pop("without_transform")

        _failures = d.pop("failures", UNSET)
        failures: list[TemplateFailureItem] | Unset = UNSET
        if _failures is not UNSET:
            failures = []
            for failures_item_data in _failures:
                failures_item = TemplateFailureItem.from_dict(failures_item_data)

                failures.append(failures_item)

        paused = cast(list[str], d.pop("paused", UNSET))

        template_check_item = cls(
            catch_all=catch_all,
            passed=passed,
            without_transform=without_transform,
            failures=failures,
            paused=paused,
        )

        return template_check_item
