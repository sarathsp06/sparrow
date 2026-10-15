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
        failures (list[TemplateFailureItem] | None | Unset): Subscriptions whose transform no longer renders against the
            new version.
        paused (list[str] | None | Unset): Subscriptions this import paused (subscription_policy=pause).
    """

    catch_all: int
    passed: int
    without_transform: int
    failures: list[TemplateFailureItem] | None | Unset = UNSET
    paused: list[str] | None | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        catch_all = self.catch_all

        passed = self.passed

        without_transform = self.without_transform

        failures: list[dict[str, Any]] | None | Unset
        if isinstance(self.failures, Unset):
            failures = UNSET
        elif isinstance(self.failures, list):
            failures = []
            for failures_type_0_item_data in self.failures:
                failures_type_0_item = failures_type_0_item_data.to_dict()
                failures.append(failures_type_0_item)

        else:
            failures = self.failures

        paused: list[str] | None | Unset
        if isinstance(self.paused, Unset):
            paused = UNSET
        elif isinstance(self.paused, list):
            paused = self.paused

        else:
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

        def _parse_failures(data: object) -> list[TemplateFailureItem] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                failures_type_0 = []
                _failures_type_0 = data
                for failures_type_0_item_data in _failures_type_0:
                    failures_type_0_item = TemplateFailureItem.from_dict(failures_type_0_item_data)

                    failures_type_0.append(failures_type_0_item)

                return failures_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[TemplateFailureItem] | None | Unset, data)

        failures = _parse_failures(d.pop("failures", UNSET))

        def _parse_paused(data: object) -> list[str] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                paused_type_0 = cast(list[str], data)

                return paused_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None | Unset, data)

        paused = _parse_paused(d.pop("paused", UNSET))

        template_check_item = cls(
            catch_all=catch_all,
            passed=passed,
            without_transform=without_transform,
            failures=failures,
            paused=paused,
        )

        return template_check_item
