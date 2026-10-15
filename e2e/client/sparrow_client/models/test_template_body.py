from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..models.test_template_body_template_missing_key import TestTemplateBodyTemplateMissingKey
from ..types import UNSET, Unset

T = TypeVar("T", bound="TestTemplateBody")


@_attrs_define
class TestTemplateBody:
    """
    Attributes:
        event_name (str): Registered event type whose sample payload the template is rendered against.
        template (str): Go template to render, in the same syntax used by transform_template.
        schema (str | Unset): A URL to the JSON Schema for this object.
        template_missing_key (TestTemplateBodyTemplateMissingKey | Unset): Render as a subscription with this
            template_missing_key would. Defaults to error, the subscription default.
    """

    event_name: str
    template: str
    schema: str | Unset = UNSET
    template_missing_key: TestTemplateBodyTemplateMissingKey | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        event_name = self.event_name

        template = self.template

        schema = self.schema

        template_missing_key: str | Unset = UNSET
        if not isinstance(self.template_missing_key, Unset):
            template_missing_key = self.template_missing_key.value

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "event_name": event_name,
                "template": template,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if template_missing_key is not UNSET:
            field_dict["template_missing_key"] = template_missing_key

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        event_name = d.pop("event_name")

        template = d.pop("template")

        schema = d.pop("$schema", UNSET)

        _template_missing_key = d.pop("template_missing_key", UNSET)
        template_missing_key: TestTemplateBodyTemplateMissingKey | Unset
        if isinstance(_template_missing_key, Unset):
            template_missing_key = UNSET
        else:
            template_missing_key = TestTemplateBodyTemplateMissingKey(_template_missing_key)

        test_template_body = cls(
            event_name=event_name,
            template=template,
            schema=schema,
            template_missing_key=template_missing_key,
        )

        return test_template_body
