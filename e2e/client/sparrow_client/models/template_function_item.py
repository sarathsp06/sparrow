from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="TemplateFunctionItem")


@_attrs_define
class TemplateFunctionItem:
    """
    Attributes:
        description (str): Markdown documentation for the function, including usage and an example.
        name (str): Function name, as used in a transform_template.
    """

    description: str
    name: str

    def to_dict(self) -> dict[str, Any]:
        description = self.description

        name = self.name

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "description": description,
                "name": name,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        description = d.pop("description")

        name = d.pop("name")

        template_function_item = cls(
            description=description,
            name=name,
        )

        return template_function_item
