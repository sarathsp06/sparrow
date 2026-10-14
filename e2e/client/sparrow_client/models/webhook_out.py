from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define

from ..models.webhook_out_health import WebhookOutHealth
from ..models.webhook_out_signature_type import WebhookOutSignatureType
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.webhook_http_config_out import WebhookHTTPConfigOut
    from ..models.webhook_out_headers import WebhookOutHeaders
    from ..models.webhook_out_secret_headers import WebhookOutSecretHeaders


T = TypeVar("T", bound="WebhookOut")


@_attrs_define
class WebhookOut:
    """
    Attributes:
        active (bool): Whether the webhook currently receives deliveries.
        consumer (str): Tenant consumer this webhook belongs to.
        created_at (str): Creation timestamp, RFC3339.
        events (list[str] | None): Event type names this webhook is currently subscribed to, derived from its
            subscriptions.
        health (WebhookOutHealth): Computed rolling health status.
        http_config (WebhookHTTPConfigOut):
        requires_transform (bool): Whether every subscription must carry an enabled transform_template because the
            receiver only accepts transformed payloads.
        signature_type (WebhookOutSignatureType): Signing scheme. "hmac" (default) signs every delivery with HMAC-SHA256
            (v1,); "ed25519" adds an Ed25519 signature (v1a,) alongside the HMAC one.
        updated_at (str): Last-modified timestamp, RFC3339.
        url (str): HTTP endpoint deliveries are POSTed to.
        webhook_id (str): Webhook id (UUID).
        schema (str | Unset): A URL to the JSON Schema for this object.
        auto_disabled_at (str | Unset): Set when Sparrow paused this webhook itself because its receiver kept failing
            (see SPARROW_AUTO_DISABLE_AFTER), RFC3339. Cleared when the webhook is resumed.
        auto_disabled_reason (str | Unset): Why Sparrow auto-disabled this webhook. Cleared when the webhook is resumed.
        description (str | Unset): Human-readable note about this webhook.
        headers (WebhookOutHeaders | Unset): Static HTTP headers sent with every delivery.
        secret_headers (WebhookOutSecretHeaders | Unset): Encrypted secret header names, with values always masked.
            Update them via PATCH secret_headers; send an empty string to remove one.
        signing_public_key (str | Unset): Hex-encoded Ed25519 public key for verifying the v1a, delivery signature. Safe
            to expose; there is no private-key equivalent to mask.
    """

    active: bool
    consumer: str
    created_at: str
    events: list[str] | None
    health: WebhookOutHealth
    http_config: WebhookHTTPConfigOut
    requires_transform: bool
    signature_type: WebhookOutSignatureType
    updated_at: str
    url: str
    webhook_id: str
    schema: str | Unset = UNSET
    auto_disabled_at: str | Unset = UNSET
    auto_disabled_reason: str | Unset = UNSET
    description: str | Unset = UNSET
    headers: WebhookOutHeaders | Unset = UNSET
    secret_headers: WebhookOutSecretHeaders | Unset = UNSET
    signing_public_key: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        active = self.active

        consumer = self.consumer

        created_at = self.created_at

        events: list[str] | None
        if isinstance(self.events, list):
            events = self.events

        else:
            events = self.events

        health = self.health.value

        http_config = self.http_config.to_dict()

        requires_transform = self.requires_transform

        signature_type = self.signature_type.value

        updated_at = self.updated_at

        url = self.url

        webhook_id = self.webhook_id

        schema = self.schema

        auto_disabled_at = self.auto_disabled_at

        auto_disabled_reason = self.auto_disabled_reason

        description = self.description

        headers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.headers, Unset):
            headers = self.headers.to_dict()

        secret_headers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.secret_headers, Unset):
            secret_headers = self.secret_headers.to_dict()

        signing_public_key = self.signing_public_key

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "active": active,
                "consumer": consumer,
                "created_at": created_at,
                "events": events,
                "health": health,
                "http_config": http_config,
                "requires_transform": requires_transform,
                "signature_type": signature_type,
                "updated_at": updated_at,
                "url": url,
                "webhook_id": webhook_id,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if auto_disabled_at is not UNSET:
            field_dict["auto_disabled_at"] = auto_disabled_at
        if auto_disabled_reason is not UNSET:
            field_dict["auto_disabled_reason"] = auto_disabled_reason
        if description is not UNSET:
            field_dict["description"] = description
        if headers is not UNSET:
            field_dict["headers"] = headers
        if secret_headers is not UNSET:
            field_dict["secret_headers"] = secret_headers
        if signing_public_key is not UNSET:
            field_dict["signing_public_key"] = signing_public_key

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.webhook_http_config_out import WebhookHTTPConfigOut  # noqa: PLC0415
        from ..models.webhook_out_headers import WebhookOutHeaders  # noqa: PLC0415
        from ..models.webhook_out_secret_headers import WebhookOutSecretHeaders  # noqa: PLC0415

        d = dict(src_dict)
        active = d.pop("active")

        consumer = d.pop("consumer")

        created_at = d.pop("created_at")

        def _parse_events(data: object) -> list[str] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                events_type_0 = cast(list[str], data)

                return events_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None, data)

        events = _parse_events(d.pop("events"))

        health = WebhookOutHealth(d.pop("health"))

        http_config = WebhookHTTPConfigOut.from_dict(d.pop("http_config"))

        requires_transform = d.pop("requires_transform")

        signature_type = WebhookOutSignatureType(d.pop("signature_type"))

        updated_at = d.pop("updated_at")

        url = d.pop("url")

        webhook_id = d.pop("webhook_id")

        schema = d.pop("$schema", UNSET)

        auto_disabled_at = d.pop("auto_disabled_at", UNSET)

        auto_disabled_reason = d.pop("auto_disabled_reason", UNSET)

        description = d.pop("description", UNSET)

        _headers = d.pop("headers", UNSET)
        headers: WebhookOutHeaders | Unset
        if isinstance(_headers, Unset):
            headers = UNSET
        else:
            headers = WebhookOutHeaders.from_dict(_headers)

        _secret_headers = d.pop("secret_headers", UNSET)
        secret_headers: WebhookOutSecretHeaders | Unset
        if isinstance(_secret_headers, Unset):
            secret_headers = UNSET
        else:
            secret_headers = WebhookOutSecretHeaders.from_dict(_secret_headers)

        signing_public_key = d.pop("signing_public_key", UNSET)

        webhook_out = cls(
            active=active,
            consumer=consumer,
            created_at=created_at,
            events=events,
            health=health,
            http_config=http_config,
            requires_transform=requires_transform,
            signature_type=signature_type,
            updated_at=updated_at,
            url=url,
            webhook_id=webhook_id,
            schema=schema,
            auto_disabled_at=auto_disabled_at,
            auto_disabled_reason=auto_disabled_reason,
            description=description,
            headers=headers,
            secret_headers=secret_headers,
            signing_public_key=signing_public_key,
        )

        return webhook_out
