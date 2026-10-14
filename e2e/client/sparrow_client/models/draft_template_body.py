from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.draft_template_body_sample_payload import DraftTemplateBodySamplePayload


T = TypeVar("T", bound="DraftTemplateBody")


@_attrs_define
class DraftTemplateBody:
    """
    Attributes:
        event_name (str): Registered event type whose schema and sample payload ground the draft, and against which it
            is verified.
        instructions (str): Plain-language description of the body the receiver should get, e.g. "a short Slack message
            with the order id and total, flagged when refunded".
        schema (str | Unset): A URL to the JSON Schema for this object.
        current_template (str | Unset): Optional existing template to refine instead of drafting from scratch.
        docs_url (str | Unset): Optional http(s) page documenting the receiver's payload. Sparrow fetches it (subject to
            the same network policy as webhook deliveries, so private and cloud-metadata addresses are refused unless
            allowed) and gives the model an excerpt of its text.
        recipe (str | Unset): Name of a shipped recipe (see GET /v1/recipes) whose destination format the draft should
            follow, e.g. "slack". Omit for a custom receiver.
        sample_payload (DraftTemplateBodySamplePayload | Unset): Optional payload to ground and verify the draft against
            instead of the event type's registered sample. Does not change the event type.
        target_example (str | Unset): Optional: what the receiver expects, either an example body (the draft maps event
            fields onto that exact structure) or a plain-language description of its shape.
    """

    event_name: str
    instructions: str
    schema: str | Unset = UNSET
    current_template: str | Unset = UNSET
    docs_url: str | Unset = UNSET
    recipe: str | Unset = UNSET
    sample_payload: DraftTemplateBodySamplePayload | Unset = UNSET
    target_example: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        event_name = self.event_name

        instructions = self.instructions

        schema = self.schema

        current_template = self.current_template

        docs_url = self.docs_url

        recipe = self.recipe

        sample_payload: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sample_payload, Unset):
            sample_payload = self.sample_payload.to_dict()

        target_example = self.target_example

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "event_name": event_name,
                "instructions": instructions,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if current_template is not UNSET:
            field_dict["current_template"] = current_template
        if docs_url is not UNSET:
            field_dict["docs_url"] = docs_url
        if recipe is not UNSET:
            field_dict["recipe"] = recipe
        if sample_payload is not UNSET:
            field_dict["sample_payload"] = sample_payload
        if target_example is not UNSET:
            field_dict["target_example"] = target_example

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.draft_template_body_sample_payload import DraftTemplateBodySamplePayload  # noqa: PLC0415

        d = dict(src_dict)
        event_name = d.pop("event_name")

        instructions = d.pop("instructions")

        schema = d.pop("$schema", UNSET)

        current_template = d.pop("current_template", UNSET)

        docs_url = d.pop("docs_url", UNSET)

        recipe = d.pop("recipe", UNSET)

        _sample_payload = d.pop("sample_payload", UNSET)
        sample_payload: DraftTemplateBodySamplePayload | Unset
        if isinstance(_sample_payload, Unset):
            sample_payload = UNSET
        else:
            sample_payload = DraftTemplateBodySamplePayload.from_dict(_sample_payload)

        target_example = d.pop("target_example", UNSET)

        draft_template_body = cls(
            event_name=event_name,
            instructions=instructions,
            schema=schema,
            current_template=current_template,
            docs_url=docs_url,
            recipe=recipe,
            sample_payload=sample_payload,
            target_example=target_example,
        )

        return draft_template_body
