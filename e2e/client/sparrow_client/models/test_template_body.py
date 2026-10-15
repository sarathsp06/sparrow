from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..models.test_template_body_template_missing_key import TestTemplateBodyTemplateMissingKey
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.test_template_body_payload import TestTemplateBodyPayload


T = TypeVar("T", bound="TestTemplateBody")


@_attrs_define
class TestTemplateBody:
    """
    Attributes:
        event_name (str): Registered event type. Its name is the envelope's event_name, and its sample_payload
            (generated from its schema; empty without one) is rendered against when payload is omitted.
        template (str): Go template to render, in the same syntax used by transform_template.
        schema (str | Unset): A URL to the JSON Schema for this object.
        payload (TestTemplateBodyPayload | Unset): Payload to render against instead of the event type's generated
            sample, e.g. a real event's payload from listEventOccurrences. Used as given, even when empty; the event type
            needs no schema.
        template_missing_key (TestTemplateBodyTemplateMissingKey | Unset): Render as a subscription with this
            template_missing_key would. Defaults to error, the subscription default.
    """

    event_name: str
    template: str
    schema: str | Unset = UNSET
    payload: TestTemplateBodyPayload | Unset = UNSET
    template_missing_key: TestTemplateBodyTemplateMissingKey | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        event_name = self.event_name

        template = self.template

        schema = self.schema

        payload: dict[str, Any] | Unset = UNSET
        if not isinstance(self.payload, Unset):
            payload = self.payload.to_dict()

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
        if payload is not UNSET:
            field_dict["payload"] = payload
        if template_missing_key is not UNSET:
            field_dict["template_missing_key"] = template_missing_key

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.test_template_body_payload import TestTemplateBodyPayload  # noqa: PLC0415

        d = dict(src_dict)
        event_name = d.pop("event_name")

        template = d.pop("template")

        schema = d.pop("$schema", UNSET)

        _payload = d.pop("payload", UNSET)
        payload: TestTemplateBodyPayload | Unset
        if isinstance(_payload, Unset):
            payload = UNSET
        else:
            payload = TestTemplateBodyPayload.from_dict(_payload)

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
            payload=payload,
            template_missing_key=template_missing_key,
        )

        return test_template_body
