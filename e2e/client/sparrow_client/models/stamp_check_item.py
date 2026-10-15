from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..models.stamp_check_item_status import StampCheckItemStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="StampCheckItem")


@_attrs_define
class StampCheckItem:
    """
    Attributes:
        server (str): This server's Sparrow version.
        status (StampCheckItemStatus): valid: stamp present and matching. unsigned: no stamp (hand-written file).
            warning: see warnings.
        exported_by (str | Unset): Sparrow version that exported the file.
        warnings (list[str] | Unset): version_differs, format_unsupported, items_changed. Each must be acknowledged for
            the import to write.
    """

    server: str
    status: StampCheckItemStatus
    exported_by: str | Unset = UNSET
    warnings: list[str] | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        server = self.server

        status = self.status.value

        exported_by = self.exported_by

        warnings: list[str] | Unset = UNSET
        if not isinstance(self.warnings, Unset):
            warnings = self.warnings

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "server": server,
                "status": status,
            }
        )
        if exported_by is not UNSET:
            field_dict["exported_by"] = exported_by
        if warnings is not UNSET:
            field_dict["warnings"] = warnings

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        server = d.pop("server")

        status = StampCheckItemStatus(d.pop("status"))

        exported_by = d.pop("exported_by", UNSET)

        warnings = cast(list[str], d.pop("warnings", UNSET))

        stamp_check_item = cls(
            server=server,
            status=status,
            exported_by=exported_by,
            warnings=warnings,
        )

        return stamp_check_item
