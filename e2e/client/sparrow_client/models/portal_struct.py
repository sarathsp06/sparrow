from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="PortalStruct")


@_attrs_define
class PortalStruct:
    """
    Attributes:
        retry (bool): True when portal tokens may retry their deliveries (SPARROW_PORTAL_RETRY, default true). When
            false, the retry calls under /portal/api/ answer 403; admins can still retry.
    """

    retry: bool

    def to_dict(self) -> dict[str, Any]:
        retry = self.retry

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "retry": retry,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        retry = d.pop("retry")

        portal_struct = cls(
            retry=retry,
        )

        return portal_struct
