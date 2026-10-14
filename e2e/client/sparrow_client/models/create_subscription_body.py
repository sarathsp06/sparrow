from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..models.create_subscription_body_on_transform_error import CreateSubscriptionBodyOnTransformError
from ..models.create_subscription_body_template_missing_key import CreateSubscriptionBodyTemplateMissingKey
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.create_subscription_body_headers import CreateSubscriptionBodyHeaders
    from ..models.create_subscription_body_label_filters import CreateSubscriptionBodyLabelFilters


T = TypeVar("T", bound="CreateSubscriptionBody")


@_attrs_define
class CreateSubscriptionBody:
    """
    Attributes:
        event_name (str): Event type name to subscribe to, or "*" to receive every event in the consumer (catch-all).
        webhook_id (str): Webhook to deliver matching events to.
        schema (str | Unset): A URL to the JSON Schema for this object.
        headers (CreateSubscriptionBodyHeaders | Unset): Extra HTTP headers to send with deliveries created by this
            subscription, merged with the webhook's own headers.
        label_filters (CreateSubscriptionBodyLabelFilters | Unset): Key/value pairs that must ALL be present in an
            event's labels for this subscription to receive it. Empty means match every event of event_name.
        method (str | Unset): HTTP method used for deliveries from this subscription. Defaults to POST.
        on_transform_error (CreateSubscriptionBodyOnTransformError | Unset): What happens when transform_template fails
            to render. fail (default): the delivery fails with error category template_error, is not retried automatically,
            and can be retried once the template is fixed; it does not affect the webhook's health. fallback: the default
            envelope payload is sent instead, and the error is still recorded on the delivery.
        template_missing_key (CreateSubscriptionBodyTemplateMissingKey | Unset): How transform_template reads a key the
            payload does not have. error (default): the render fails, so a field removed from the event schema cannot
            silently turn into "<no value>" in the body; read optional fields with index or dig (default only fills a key
            that is present but empty). zero: the key renders as "<no value>".
        timeout (int | Unset): Per-delivery request timeout in seconds, overriding the webhook's default.
        transform_enabled (bool | Unset): Whether to render transform_template into the delivered payload instead of
            sending the raw event payload.
        transform_template (str | Unset): Go template rendered against the event to produce the delivered body. See GET
            /v1/template-functions for available helpers; test it with POST /v1/subscriptions:testTemplate.
    """

    event_name: str
    webhook_id: str
    schema: str | Unset = UNSET
    headers: CreateSubscriptionBodyHeaders | Unset = UNSET
    label_filters: CreateSubscriptionBodyLabelFilters | Unset = UNSET
    method: str | Unset = UNSET
    on_transform_error: CreateSubscriptionBodyOnTransformError | Unset = UNSET
    template_missing_key: CreateSubscriptionBodyTemplateMissingKey | Unset = UNSET
    timeout: int | Unset = UNSET
    transform_enabled: bool | Unset = UNSET
    transform_template: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        event_name = self.event_name

        webhook_id = self.webhook_id

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

        timeout = self.timeout

        transform_enabled = self.transform_enabled

        transform_template = self.transform_template

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "event_name": event_name,
                "webhook_id": webhook_id,
            }
        )
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
        if timeout is not UNSET:
            field_dict["timeout"] = timeout
        if transform_enabled is not UNSET:
            field_dict["transform_enabled"] = transform_enabled
        if transform_template is not UNSET:
            field_dict["transform_template"] = transform_template

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.create_subscription_body_headers import CreateSubscriptionBodyHeaders  # noqa: PLC0415
        from ..models.create_subscription_body_label_filters import CreateSubscriptionBodyLabelFilters  # noqa: PLC0415

        d = dict(src_dict)
        event_name = d.pop("event_name")

        webhook_id = d.pop("webhook_id")

        schema = d.pop("$schema", UNSET)

        _headers = d.pop("headers", UNSET)
        headers: CreateSubscriptionBodyHeaders | Unset
        if isinstance(_headers, Unset):
            headers = UNSET
        else:
            headers = CreateSubscriptionBodyHeaders.from_dict(_headers)

        _label_filters = d.pop("label_filters", UNSET)
        label_filters: CreateSubscriptionBodyLabelFilters | Unset
        if isinstance(_label_filters, Unset):
            label_filters = UNSET
        else:
            label_filters = CreateSubscriptionBodyLabelFilters.from_dict(_label_filters)

        method = d.pop("method", UNSET)

        _on_transform_error = d.pop("on_transform_error", UNSET)
        on_transform_error: CreateSubscriptionBodyOnTransformError | Unset
        if isinstance(_on_transform_error, Unset):
            on_transform_error = UNSET
        else:
            on_transform_error = CreateSubscriptionBodyOnTransformError(_on_transform_error)

        _template_missing_key = d.pop("template_missing_key", UNSET)
        template_missing_key: CreateSubscriptionBodyTemplateMissingKey | Unset
        if isinstance(_template_missing_key, Unset):
            template_missing_key = UNSET
        else:
            template_missing_key = CreateSubscriptionBodyTemplateMissingKey(_template_missing_key)

        timeout = d.pop("timeout", UNSET)

        transform_enabled = d.pop("transform_enabled", UNSET)

        transform_template = d.pop("transform_template", UNSET)

        create_subscription_body = cls(
            event_name=event_name,
            webhook_id=webhook_id,
            schema=schema,
            headers=headers,
            label_filters=label_filters,
            method=method,
            on_transform_error=on_transform_error,
            template_missing_key=template_missing_key,
            timeout=timeout,
            transform_enabled=transform_enabled,
            transform_template=transform_template,
        )

        return create_subscription_body
