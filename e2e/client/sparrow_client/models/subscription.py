from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="Subscription")


@_attrs_define
class Subscription:
    """
    Attributes:
        transform_template (str): Go template rendered per delivery to build the destination-native body; may contain
            {{param}} tokens.
    """

    transform_template: str

    def to_dict(self) -> dict[str, Any]:
        transform_template = self.transform_template

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "transform_template": transform_template,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        transform_template = d.pop("transform_template")

        subscription = cls(
            transform_template=transform_template,
        )

        return subscription
