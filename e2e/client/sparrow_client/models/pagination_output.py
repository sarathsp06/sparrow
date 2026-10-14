from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="PaginationOutput")


@_attrs_define
class PaginationOutput:
    """
    Attributes:
        has_more (bool):
        limit (int):
        offset (int):
        total_count (int):
    """

    has_more: bool
    limit: int
    offset: int
    total_count: int

    def to_dict(self) -> dict[str, Any]:
        has_more = self.has_more

        limit = self.limit

        offset = self.offset

        total_count = self.total_count

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "has_more": has_more,
                "limit": limit,
                "offset": offset,
                "total_count": total_count,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        has_more = d.pop("has_more")

        limit = d.pop("limit")

        offset = d.pop("offset")

        total_count = d.pop("total_count")

        pagination_output = cls(
            has_more=has_more,
            limit=limit,
            offset=offset,
            total_count=total_count,
        )

        return pagination_output
