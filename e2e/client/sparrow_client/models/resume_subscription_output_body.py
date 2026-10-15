from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..models.resume_subscription_output_body_method import ResumeSubscriptionOutputBodyMethod
from ..models.resume_subscription_output_body_on_transform_error import ResumeSubscriptionOutputBodyOnTransformError
from ..models.resume_subscription_output_body_template_missing_key import ResumeSubscriptionOutputBodyTemplateMissingKey
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.resume_subscription_output_body_headers import ResumeSubscriptionOutputBodyHeaders
    from ..models.resume_subscription_output_body_label_filters import ResumeSubscriptionOutputBodyLabelFilters


T = TypeVar("T", bound="ResumeSubscriptionOutputBody")


@_attrs_define
class ResumeSubscriptionOutputBody:
    """
    Attributes:
        consumer (str): Consumer this subscription belongs to.
        created_at (str): Creation timestamp, RFC3339.
        event_name (str): Event type name this subscription matches, or "*" for catch-all.
        on_transform_error (ResumeSubscriptionOutputBodyOnTransformError): What happens when transform_template fails to
            render. fail (default): the delivery fails with error category template_error, is not retried automatically, and
            can be retried once the template is fixed; it does not affect the webhook's health. fallback: the default
            envelope payload is sent instead, and the error is still recorded on the delivery.
        paused (bool): Whether the subscription is paused. While paused, events still fan out to it, but each delivery
            is recorded with status paused and not attempted until retried. A pause never affects the webhook's health.
        paused_deliveries (int): Deliveries of this subscription currently in status paused. They are not sent
            automatically; retry them to deliver.
        subscription_id (str): Subscription id (UUID).
        template_missing_key (ResumeSubscriptionOutputBodyTemplateMissingKey): How transform_template reads a key the
            payload does not have. error (default): the render fails, so a field removed from the event schema cannot
            silently turn into "<no value>" in the body; read optional fields with index or dig (default only fills a key
            that is present but empty). zero: the key renders as "<no value>".
        transform_enabled (bool): Whether transform_template is rendered into the delivered payload.
        updated_at (str): Last-modified timestamp, RFC3339.
        webhook_id (str): Webhook this subscription delivers to.
        schema (str | Unset): A URL to the JSON Schema for this object.
        headers (ResumeSubscriptionOutputBodyHeaders | Unset): Extra HTTP headers sent with deliveries from this
            subscription.
        label_filters (ResumeSubscriptionOutputBodyLabelFilters | Unset): Key/value pairs that must all be present in an
            event's labels for this subscription to receive it.
        method (ResumeSubscriptionOutputBodyMethod | Unset): HTTP method used for deliveries from this subscription;
            absent means POST.
        paused_at (str | Unset): When the current pause began, RFC3339. Absent when not paused.
        paused_reason (str | Unset): Why the subscription was paused.
        paused_since (str | Unset): When the pause that just ended began, RFC3339; absent when the subscription was not
            paused. List or retry what was held with status=paused&subscription_id=...&created_after=<paused_since>.
        timeout (int | Unset): Request timeout override for this subscription's deliveries, in seconds; absent means the
            webhook's request_timeout_seconds.
        transform_template (str | Unset): Go text/template rendered against the event to produce the delivered body when
            transform_enabled is true.
    """

    consumer: str
    created_at: str
    event_name: str
    on_transform_error: ResumeSubscriptionOutputBodyOnTransformError
    paused: bool
    paused_deliveries: int
    subscription_id: str
    template_missing_key: ResumeSubscriptionOutputBodyTemplateMissingKey
    transform_enabled: bool
    updated_at: str
    webhook_id: str
    schema: str | Unset = UNSET
    headers: ResumeSubscriptionOutputBodyHeaders | Unset = UNSET
    label_filters: ResumeSubscriptionOutputBodyLabelFilters | Unset = UNSET
    method: ResumeSubscriptionOutputBodyMethod | Unset = UNSET
    paused_at: str | Unset = UNSET
    paused_reason: str | Unset = UNSET
    paused_since: str | Unset = UNSET
    timeout: int | Unset = UNSET
    transform_template: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        consumer = self.consumer

        created_at = self.created_at

        event_name = self.event_name

        on_transform_error = self.on_transform_error.value

        paused = self.paused

        paused_deliveries = self.paused_deliveries

        subscription_id = self.subscription_id

        template_missing_key = self.template_missing_key.value

        transform_enabled = self.transform_enabled

        updated_at = self.updated_at

        webhook_id = self.webhook_id

        schema = self.schema

        headers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.headers, Unset):
            headers = self.headers.to_dict()

        label_filters: dict[str, Any] | Unset = UNSET
        if not isinstance(self.label_filters, Unset):
            label_filters = self.label_filters.to_dict()

        method: str | Unset = UNSET
        if not isinstance(self.method, Unset):
            method = self.method.value

        paused_at = self.paused_at

        paused_reason = self.paused_reason

        paused_since = self.paused_since

        timeout = self.timeout

        transform_template = self.transform_template

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "consumer": consumer,
                "created_at": created_at,
                "event_name": event_name,
                "on_transform_error": on_transform_error,
                "paused": paused,
                "paused_deliveries": paused_deliveries,
                "subscription_id": subscription_id,
                "template_missing_key": template_missing_key,
                "transform_enabled": transform_enabled,
                "updated_at": updated_at,
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
        if paused_at is not UNSET:
            field_dict["paused_at"] = paused_at
        if paused_reason is not UNSET:
            field_dict["paused_reason"] = paused_reason
        if paused_since is not UNSET:
            field_dict["paused_since"] = paused_since
        if timeout is not UNSET:
            field_dict["timeout"] = timeout
        if transform_template is not UNSET:
            field_dict["transform_template"] = transform_template

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.resume_subscription_output_body_headers import ResumeSubscriptionOutputBodyHeaders  # noqa: PLC0415
        from ..models.resume_subscription_output_body_label_filters import ResumeSubscriptionOutputBodyLabelFilters  # noqa: PLC0415

        d = dict(src_dict)
        consumer = d.pop("consumer")

        created_at = d.pop("created_at")

        event_name = d.pop("event_name")

        on_transform_error = ResumeSubscriptionOutputBodyOnTransformError(d.pop("on_transform_error"))

        paused = d.pop("paused")

        paused_deliveries = d.pop("paused_deliveries")

        subscription_id = d.pop("subscription_id")

        template_missing_key = ResumeSubscriptionOutputBodyTemplateMissingKey(d.pop("template_missing_key"))

        transform_enabled = d.pop("transform_enabled")

        updated_at = d.pop("updated_at")

        webhook_id = d.pop("webhook_id")

        schema = d.pop("$schema", UNSET)

        _headers = d.pop("headers", UNSET)
        headers: ResumeSubscriptionOutputBodyHeaders | Unset
        if isinstance(_headers, Unset):
            headers = UNSET
        else:
            headers = ResumeSubscriptionOutputBodyHeaders.from_dict(_headers)

        _label_filters = d.pop("label_filters", UNSET)
        label_filters: ResumeSubscriptionOutputBodyLabelFilters | Unset
        if isinstance(_label_filters, Unset):
            label_filters = UNSET
        else:
            label_filters = ResumeSubscriptionOutputBodyLabelFilters.from_dict(_label_filters)

        _method = d.pop("method", UNSET)
        method: ResumeSubscriptionOutputBodyMethod | Unset
        if isinstance(_method, Unset):
            method = UNSET
        else:
            method = ResumeSubscriptionOutputBodyMethod(_method)

        paused_at = d.pop("paused_at", UNSET)

        paused_reason = d.pop("paused_reason", UNSET)

        paused_since = d.pop("paused_since", UNSET)

        timeout = d.pop("timeout", UNSET)

        transform_template = d.pop("transform_template", UNSET)

        resume_subscription_output_body = cls(
            consumer=consumer,
            created_at=created_at,
            event_name=event_name,
            on_transform_error=on_transform_error,
            paused=paused,
            paused_deliveries=paused_deliveries,
            subscription_id=subscription_id,
            template_missing_key=template_missing_key,
            transform_enabled=transform_enabled,
            updated_at=updated_at,
            webhook_id=webhook_id,
            schema=schema,
            headers=headers,
            label_filters=label_filters,
            method=method,
            paused_at=paused_at,
            paused_reason=paused_reason,
            paused_since=paused_since,
            timeout=timeout,
            transform_template=transform_template,
        )

        return resume_subscription_output_body
