from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.patch_webhook_body_headers import PatchWebhookBodyHeaders
    from ..models.patch_webhook_body_secret_headers import PatchWebhookBodySecretHeaders
    from ..models.webhook_http_config import WebhookHTTPConfig


T = TypeVar("T", bound="PatchWebhookBody")


@_attrs_define
class PatchWebhookBody:
    """
    Attributes:
        schema (str | Unset): A URL to the JSON Schema for this object.
        active (bool | Unset): false pauses the webhook (like :pause), true resumes it and clears any auto-disable
            marker (like :resume, but without returning the held count).
        description (str | Unset): Replace the human-readable description. An empty string leaves it unchanged.
        events (list[str] | Unset): Replace the full set of subscribed event type names. Every existing subscription of
            the webhook is deleted, with its transform and label filters, and one plain subscription is created per name. An
            empty list leaves subscriptions unchanged. Rejected with 409 while requires_transform is on; to keep transforms,
            add or delete subscriptions individually.
        headers (PatchWebhookBodyHeaders | Unset): Replace all static headers sent with every delivery. Same rules as on
            register.
        http_config (WebhookHTTPConfig | Unset):
        requires_transform (bool | Unset): Require (or stop requiring) a payload transform on every subscription.
            Turning it on fails with 409 while any subscription has no enabled transform_template. While on, events cannot
            be replaced in bulk (409): add or delete subscriptions individually.
        secret_headers (PatchWebhookBodySecretHeaders | Unset): Merge-patch encrypted secret headers by name. Send a new
            value to replace one header, omit a key to leave it untouched, or send an empty string to remove it. An empty
            object removes them all.
        signature_type (str | Unset): Change the signing scheme: hmac or ed25519. Switching to ed25519 generates a
            signing keypair; HMAC signing continues either way.
        timeout (int | Unset): Deprecated: use http_config.request_timeout_seconds. Shorthand for it, in seconds (1 to
            300), applied only when http_config.request_timeout_seconds is not sent (http_config wins when both are sent).
        url (str | Unset): Replace the delivery endpoint URL. Same rules as on register.
    """

    schema: str | Unset = UNSET
    active: bool | Unset = UNSET
    description: str | Unset = UNSET
    events: list[str] | Unset = UNSET
    headers: PatchWebhookBodyHeaders | Unset = UNSET
    http_config: WebhookHTTPConfig | Unset = UNSET
    requires_transform: bool | Unset = UNSET
    secret_headers: PatchWebhookBodySecretHeaders | Unset = UNSET
    signature_type: str | Unset = UNSET
    timeout: int | Unset = UNSET
    url: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        schema = self.schema

        active = self.active

        description = self.description

        events: list[str] | Unset = UNSET
        if not isinstance(self.events, Unset):
            events = self.events

        headers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.headers, Unset):
            headers = self.headers.to_dict()

        http_config: dict[str, Any] | Unset = UNSET
        if not isinstance(self.http_config, Unset):
            http_config = self.http_config.to_dict()

        requires_transform = self.requires_transform

        secret_headers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.secret_headers, Unset):
            secret_headers = self.secret_headers.to_dict()

        signature_type = self.signature_type

        timeout = self.timeout

        url = self.url

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if active is not UNSET:
            field_dict["active"] = active
        if description is not UNSET:
            field_dict["description"] = description
        if events is not UNSET:
            field_dict["events"] = events
        if headers is not UNSET:
            field_dict["headers"] = headers
        if http_config is not UNSET:
            field_dict["http_config"] = http_config
        if requires_transform is not UNSET:
            field_dict["requires_transform"] = requires_transform
        if secret_headers is not UNSET:
            field_dict["secret_headers"] = secret_headers
        if signature_type is not UNSET:
            field_dict["signature_type"] = signature_type
        if timeout is not UNSET:
            field_dict["timeout"] = timeout
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.patch_webhook_body_headers import PatchWebhookBodyHeaders  # noqa: PLC0415
        from ..models.patch_webhook_body_secret_headers import PatchWebhookBodySecretHeaders  # noqa: PLC0415
        from ..models.webhook_http_config import WebhookHTTPConfig  # noqa: PLC0415

        d = dict(src_dict)
        schema = d.pop("$schema", UNSET)

        active = d.pop("active", UNSET)

        description = d.pop("description", UNSET)

        events = cast(list[str], d.pop("events", UNSET))

        _headers = d.pop("headers", UNSET)
        headers: PatchWebhookBodyHeaders | Unset
        if isinstance(_headers, Unset):
            headers = UNSET
        else:
            headers = PatchWebhookBodyHeaders.from_dict(_headers)

        _http_config = d.pop("http_config", UNSET)
        http_config: WebhookHTTPConfig | Unset
        if isinstance(_http_config, Unset):
            http_config = UNSET
        else:
            http_config = WebhookHTTPConfig.from_dict(_http_config)

        requires_transform = d.pop("requires_transform", UNSET)

        _secret_headers = d.pop("secret_headers", UNSET)
        secret_headers: PatchWebhookBodySecretHeaders | Unset
        if isinstance(_secret_headers, Unset):
            secret_headers = UNSET
        else:
            secret_headers = PatchWebhookBodySecretHeaders.from_dict(_secret_headers)

        signature_type = d.pop("signature_type", UNSET)

        timeout = d.pop("timeout", UNSET)

        url = d.pop("url", UNSET)

        patch_webhook_body = cls(
            schema=schema,
            active=active,
            description=description,
            events=events,
            headers=headers,
            http_config=http_config,
            requires_transform=requires_transform,
            secret_headers=secret_headers,
            signature_type=signature_type,
            timeout=timeout,
            url=url,
        )

        return patch_webhook_body
