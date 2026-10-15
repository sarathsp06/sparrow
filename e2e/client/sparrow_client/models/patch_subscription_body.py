from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..models.patch_subscription_body_on_transform_error import PatchSubscriptionBodyOnTransformError
from ..models.patch_subscription_body_template_missing_key import PatchSubscriptionBodyTemplateMissingKey
from ..models.patch_subscription_body_template_source import PatchSubscriptionBodyTemplateSource
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.patch_subscription_body_headers import PatchSubscriptionBodyHeaders
    from ..models.patch_subscription_body_label_filters import PatchSubscriptionBodyLabelFilters


T = TypeVar("T", bound="PatchSubscriptionBody")


@_attrs_define
class PatchSubscriptionBody:
    """
    Attributes:
        schema (str | Unset): A URL to the JSON Schema for this object.
        headers (PatchSubscriptionBodyHeaders | Unset): Replace all extra HTTP headers.
        label_filters (PatchSubscriptionBodyLabelFilters | Unset): Replace all label filters; an empty object matches
            every event.
        method (str | Unset): Replace the HTTP method.
        on_transform_error (PatchSubscriptionBodyOnTransformError | Unset): Replace what happens when the template fails
            to render.
        template_missing_key (PatchSubscriptionBodyTemplateMissingKey | Unset): Replace how the template reads a missing
            key.
        template_notes (str | Unset): Optional note stored with the template version, e.g. the AI drafter's summary.
        template_source (PatchSubscriptionBodyTemplateSource | Unset): How a changed transform_template was produced,
            recorded in the subscription's template history. Defaults to manual.
        timeout (int | Unset): Replace the request timeout override, in seconds (0 to 300; 0 uses the webhook's).
        transform_enabled (bool | Unset): Enable or disable payload transformation.
        transform_template (str | Unset): Replace the transform template. A change is recorded as a new version in the
            template history (the last 20 are kept).
    """

    schema: str | Unset = UNSET
    headers: PatchSubscriptionBodyHeaders | Unset = UNSET
    label_filters: PatchSubscriptionBodyLabelFilters | Unset = UNSET
    method: str | Unset = UNSET
    on_transform_error: PatchSubscriptionBodyOnTransformError | Unset = UNSET
    template_missing_key: PatchSubscriptionBodyTemplateMissingKey | Unset = UNSET
    template_notes: str | Unset = UNSET
    template_source: PatchSubscriptionBodyTemplateSource | Unset = UNSET
    timeout: int | Unset = UNSET
    transform_enabled: bool | Unset = UNSET
    transform_template: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        schema = self.schema

        headers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.headers, Unset):
            headers = self.headers.to_dict()

        label_filters: dict[str, Any] | Unset = UNSET
        if not isinstance(self.label_filters, Unset):
            label_filters = self.label_filters.to_dict()

        method = self.method

        on_transform_error: str | Unset = UNSET
        if not isinstance(self.on_transform_error, Unset):
            on_transform_error = self.on_transform_error.value

        template_missing_key: str | Unset = UNSET
        if not isinstance(self.template_missing_key, Unset):
            template_missing_key = self.template_missing_key.value

        template_notes = self.template_notes

        template_source: str | Unset = UNSET
        if not isinstance(self.template_source, Unset):
            template_source = self.template_source.value

        timeout = self.timeout

        transform_enabled = self.transform_enabled

        transform_template = self.transform_template

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if headers is not UNSET:
            field_dict["headers"] = headers
        if label_filters is not UNSET:
            field_dict["label_filters"] = label_filters
        if method is not UNSET:
            field_dict["method"] = method
        if on_transform_error is not UNSET:
            field_dict["on_transform_error"] = on_transform_error
        if template_missing_key is not UNSET:
            field_dict["template_missing_key"] = template_missing_key
        if template_notes is not UNSET:
            field_dict["template_notes"] = template_notes
        if template_source is not UNSET:
            field_dict["template_source"] = template_source
        if timeout is not UNSET:
            field_dict["timeout"] = timeout
        if transform_enabled is not UNSET:
            field_dict["transform_enabled"] = transform_enabled
        if transform_template is not UNSET:
            field_dict["transform_template"] = transform_template

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.patch_subscription_body_headers import PatchSubscriptionBodyHeaders  # noqa: PLC0415
        from ..models.patch_subscription_body_label_filters import PatchSubscriptionBodyLabelFilters  # noqa: PLC0415

        d = dict(src_dict)
        schema = d.pop("$schema", UNSET)

        _headers = d.pop("headers", UNSET)
        headers: PatchSubscriptionBodyHeaders | Unset
        if isinstance(_headers, Unset):
            headers = UNSET
        else:
            headers = PatchSubscriptionBodyHeaders.from_dict(_headers)

        _label_filters = d.pop("label_filters", UNSET)
        label_filters: PatchSubscriptionBodyLabelFilters | Unset
        if isinstance(_label_filters, Unset):
            label_filters = UNSET
        else:
            label_filters = PatchSubscriptionBodyLabelFilters.from_dict(_label_filters)

        method = d.pop("method", UNSET)

        _on_transform_error = d.pop("on_transform_error", UNSET)
        on_transform_error: PatchSubscriptionBodyOnTransformError | Unset
        if isinstance(_on_transform_error, Unset):
            on_transform_error = UNSET
        else:
            on_transform_error = PatchSubscriptionBodyOnTransformError(_on_transform_error)

        _template_missing_key = d.pop("template_missing_key", UNSET)
        template_missing_key: PatchSubscriptionBodyTemplateMissingKey | Unset
        if isinstance(_template_missing_key, Unset):
            template_missing_key = UNSET
        else:
            template_missing_key = PatchSubscriptionBodyTemplateMissingKey(_template_missing_key)

        template_notes = d.pop("template_notes", UNSET)

        _template_source = d.pop("template_source", UNSET)
        template_source: PatchSubscriptionBodyTemplateSource | Unset
        if isinstance(_template_source, Unset):
            template_source = UNSET
        else:
            template_source = PatchSubscriptionBodyTemplateSource(_template_source)

        timeout = d.pop("timeout", UNSET)

        transform_enabled = d.pop("transform_enabled", UNSET)

        transform_template = d.pop("transform_template", UNSET)

        patch_subscription_body = cls(
            schema=schema,
            headers=headers,
            label_filters=label_filters,
            method=method,
            on_transform_error=on_transform_error,
            template_missing_key=template_missing_key,
            template_notes=template_notes,
            template_source=template_source,
            timeout=timeout,
            transform_enabled=transform_enabled,
            transform_template=transform_template,
        )

        return patch_subscription_body
