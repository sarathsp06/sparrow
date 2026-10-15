from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="BundleStampBody")


@_attrs_define
class BundleStampBody:
    """
    Attributes:
        format_ (int): Bundle format number.
        sha256 (str): sha256 of the canonical encoding of items, to tell whether they changed after export.
        sparrow_version (str): Version of the Sparrow server that exported the bundle.
    """

    format_: int
    sha256: str
    sparrow_version: str

    def to_dict(self) -> dict[str, Any]:
        format_ = self.format_

        sha256 = self.sha256

        sparrow_version = self.sparrow_version

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "format": format_,
                "sha256": sha256,
                "sparrow_version": sparrow_version,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        format_ = d.pop("format")

        sha256 = d.pop("sha256")

        sparrow_version = d.pop("sparrow_version")

        bundle_stamp_body = cls(
            format_=format_,
            sha256=sha256,
            sparrow_version=sparrow_version,
        )

        return bundle_stamp_body
