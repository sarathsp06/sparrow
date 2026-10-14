from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="ConsumerHint")


@_attrs_define
class ConsumerHint:
    """
    Attributes:
        name (str): Consumer to register the recipe under, e.g. _sparrow.
        note (str): Why, and what the recipe does under any other consumer.
    """

    name: str
    note: str

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        note = self.note

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "name": name,
                "note": note,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        note = d.pop("note")

        consumer_hint = cls(
            name=name,
            note=note,
        )

        return consumer_hint
