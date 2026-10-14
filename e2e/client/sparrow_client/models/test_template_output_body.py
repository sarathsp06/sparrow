from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="TestTemplateOutputBody")


@_attrs_define
class TestTemplateOutputBody:
    """
    Attributes:
        rendered (str): The template rendered against the event type's sample payload.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    rendered: str
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        rendered = self.rendered

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "rendered": rendered,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        rendered = d.pop("rendered")

        schema = d.pop("$schema", UNSET)

        test_template_output_body = cls(
            rendered=rendered,
            schema=schema,
        )

        return test_template_output_body
