from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="ReceiverFileOut")


@_attrs_define
class ReceiverFileOut:
    """
    Attributes:
        content (str): File content (UTF-8 text).
        mode (int): Unix permission bits, e.g. 420 (0644) or 493 (0755).
        path (str): Slash-separated path relative to the project root.
    """

    content: str
    mode: int
    path: str

    def to_dict(self) -> dict[str, Any]:
        content = self.content

        mode = self.mode

        path = self.path

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "content": content,
                "mode": mode,
                "path": path,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        content = d.pop("content")

        mode = d.pop("mode")

        path = d.pop("path")

        receiver_file_out = cls(
            content=content,
            mode=mode,
            path=path,
        )

        return receiver_file_out
