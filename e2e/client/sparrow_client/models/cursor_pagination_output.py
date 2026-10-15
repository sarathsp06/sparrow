from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="CursorPaginationOutput")


@_attrs_define
class CursorPaginationOutput:
    """
    Attributes:
        has_more (bool): True when older items follow this page.
        limit (int): Page size applied (the requested limit, default 50).
        next_cursor (str | Unset): Pass as cursor to get the next, older page. Absent on the last page.
    """

    has_more: bool
    limit: int
    next_cursor: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        has_more = self.has_more

        limit = self.limit

        next_cursor = self.next_cursor

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "has_more": has_more,
                "limit": limit,
            }
        )
        if next_cursor is not UNSET:
            field_dict["next_cursor"] = next_cursor

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        has_more = d.pop("has_more")

        limit = d.pop("limit")

        next_cursor = d.pop("next_cursor", UNSET)

        cursor_pagination_output = cls(
            has_more=has_more,
            limit=limit,
            next_cursor=next_cursor,
        )

        return cursor_pagination_output
