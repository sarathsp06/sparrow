from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..models.draft_template_prompt_output_body_sample_source import DraftTemplatePromptOutputBodySampleSource
from ..types import UNSET, Unset

T = TypeVar("T", bound="DraftTemplatePromptOutputBody")


@_attrs_define
class DraftTemplatePromptOutputBody:
    """
    Attributes:
        prompt (str): A self-contained prompt to paste into any chat assistant: the template data model, rules, helper
            catalog, the event's schema and sample payload, any recipe, receiver example, docs excerpt, current template,
            and the instructions. Asks for the template in a code block.
        sample_source (DraftTemplatePromptOutputBodySampleSource): Which sample payload the prompt carries.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    prompt: str
    sample_source: DraftTemplatePromptOutputBodySampleSource
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        prompt = self.prompt

        sample_source = self.sample_source.value

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "prompt": prompt,
                "sample_source": sample_source,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        prompt = d.pop("prompt")

        sample_source = DraftTemplatePromptOutputBodySampleSource(d.pop("sample_source"))

        schema = d.pop("$schema", UNSET)

        draft_template_prompt_output_body = cls(
            prompt=prompt,
            sample_source=sample_source,
            schema=schema,
        )

        return draft_template_prompt_output_body
