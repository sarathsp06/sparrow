from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..models.draft_template_output_body_sample_source import DraftTemplateOutputBodySampleSource
from ..types import UNSET, Unset

T = TypeVar("T", bound="DraftTemplateOutputBody")


@_attrs_define
class DraftTemplateOutputBody:
    """
    Attributes:
        attempts (int): Draft/render rounds it took; more than 1 means an earlier draft failed to render and was
            repaired.
        model (str): Model id that produced the draft.
        rendered (str): The template rendered against the sample payload used (provided or registered), proving it
            renders.
        sample_source (DraftTemplateOutputBodySampleSource): Which sample payload grounded and verified the draft:
            registered or provided.
        template (str): The drafted Go template, ready to save as transform_template.
        schema (str | Unset): A URL to the JSON Schema for this object.
        notes (str | Unset): Short explanation from the model of the fields used and assumptions made.
    """

    attempts: int
    model: str
    rendered: str
    sample_source: DraftTemplateOutputBodySampleSource
    template: str
    schema: str | Unset = UNSET
    notes: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        attempts = self.attempts

        model = self.model

        rendered = self.rendered

        sample_source = self.sample_source.value

        template = self.template

        schema = self.schema

        notes = self.notes

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "attempts": attempts,
                "model": model,
                "rendered": rendered,
                "sample_source": sample_source,
                "template": template,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if notes is not UNSET:
            field_dict["notes"] = notes

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        attempts = d.pop("attempts")

        model = d.pop("model")

        rendered = d.pop("rendered")

        sample_source = DraftTemplateOutputBodySampleSource(d.pop("sample_source"))

        template = d.pop("template")

        schema = d.pop("$schema", UNSET)

        notes = d.pop("notes", UNSET)

        draft_template_output_body = cls(
            attempts=attempts,
            model=model,
            rendered=rendered,
            sample_source=sample_source,
            template=template,
            schema=schema,
            notes=notes,
        )

        return draft_template_output_body
