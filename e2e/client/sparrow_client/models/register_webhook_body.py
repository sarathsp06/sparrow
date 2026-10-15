from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define

from ..models.register_webhook_body_on_transform_error import RegisterWebhookBodyOnTransformError
from ..models.register_webhook_body_signature_type import RegisterWebhookBodySignatureType
from ..models.register_webhook_body_template_missing_key import RegisterWebhookBodyTemplateMissingKey
from ..models.register_webhook_body_template_source import RegisterWebhookBodyTemplateSource
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.register_webhook_body_headers import RegisterWebhookBodyHeaders
    from ..models.register_webhook_body_secret_headers import RegisterWebhookBodySecretHeaders
    from ..models.webhook_http_config import WebhookHTTPConfig


T = TypeVar("T", bound="RegisterWebhookBody")


@_attrs_define
class RegisterWebhookBody:
    """
    Attributes:
        url (str): http or https endpoint deliveries are POSTed to. The host must resolve: localhost, *.internal,
            *.local, private, loopback and cloud metadata addresses are rejected (SSRF protection) unless allowed by
            SPARROW_ALLOW_PRIVATE_NETWORKS or SPARROW_ALLOWED_NETWORKS.
        schema (str | Unset): A URL to the JSON Schema for this object.
        active (bool | Unset): Whether the webhook is sent deliveries. Defaults to true. An inactive (paused) webhook
            still gets deliveries recorded for new events, held with status paused and sent only when retried.
        description (str | Unset): Free-text note for humans, e.g. which system or team owns this endpoint.
        events (list[str] | Unset): Event type names this webhook should receive; auto-creates one subscription per
            entry. Use "*" as the sole entry to subscribe to every event in the consumer. Omit or leave empty to register
            the webhook with no subscriptions, then attach them individually via POST .../subscriptions (e.g. to set a per-
            subscription transform_template).
        headers (RegisterWebhookBodyHeaders | Unset): Static HTTP headers sent with every delivery to this webhook. Only
            string values are sent. Names must be valid HTTP tokens, values at most 8 KiB with no CR/LF; framing headers
            (Host, Content-Length, Transfer-Encoding, Connection, Keep-Alive, Proxy-Connection, TE, Trailer, Upgrade) are
            rejected. Stored and returned in plaintext: put credentials in secret_headers.
        http_config (WebhookHTTPConfig | Unset):
        on_transform_error (RegisterWebhookBodyOnTransformError | Unset): What happens when transform_template fails to
            render for one of the created subscriptions. fail (default) or fallback; see the subscription's
            on_transform_error.
        rate_limit_rps (float | Unset): Deprecated: use http_config.rate_limit_rps. Shorthand for it, applied only when
            http_config.rate_limit_rps is not set (http_config wins when both are sent). Requests per second, must be
            positive.
        requires_transform (bool | Unset): Set when the receiver only accepts a transformed payload (Slack, SendGrid,
            ...). Every subscription must then have transform_enabled and a transform_template: transform_template is
            required here when events is non-empty, and later subscription writes without one are rejected. Recipes set
            this. Defaults to false.
        secret_headers (RegisterWebhookBodySecretHeaders | Unset): HTTP headers sent with every delivery whose values
            are envelope-encrypted at rest and masked in every API response (e.g. an upstream auth token). Same name and
            value rules as headers.
        signature_type (RegisterWebhookBodySignatureType | Unset): Signing scheme. Every delivery carries an HMAC-SHA256
            signature (v1,) in the Standard Webhooks webhook-signature header; ed25519 also generates a signing keypair and
            adds an Ed25519 signature (v1a,) next to it. Defaults to hmac.
        template_missing_key (RegisterWebhookBodyTemplateMissingKey | Unset): How transform_template reads a key the
            payload does not have, for the created subscriptions. error (default) or zero; see the subscription's
            template_missing_key.
        template_notes (str | Unset): Optional note stored with that first template version, e.g. the AI drafter's
            summary.
        template_source (RegisterWebhookBodyTemplateSource | Unset): How transform_template was produced, recorded as
            the first version in each created subscription's template history. Defaults to manual.
        transform_template (str | Unset): Transform template given to every subscription created from events, with
            transform_enabled set. Omit to create them untransformed (not allowed with requires_transform).
    """

    url: str
    schema: str | Unset = UNSET
    active: bool | Unset = UNSET
    description: str | Unset = UNSET
    events: list[str] | Unset = UNSET
    headers: RegisterWebhookBodyHeaders | Unset = UNSET
    http_config: WebhookHTTPConfig | Unset = UNSET
    on_transform_error: RegisterWebhookBodyOnTransformError | Unset = UNSET
    rate_limit_rps: float | Unset = UNSET
    requires_transform: bool | Unset = UNSET
    secret_headers: RegisterWebhookBodySecretHeaders | Unset = UNSET
    signature_type: RegisterWebhookBodySignatureType | Unset = UNSET
    template_missing_key: RegisterWebhookBodyTemplateMissingKey | Unset = UNSET
    template_notes: str | Unset = UNSET
    template_source: RegisterWebhookBodyTemplateSource | Unset = UNSET
    transform_template: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        url = self.url

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

        on_transform_error: str | Unset = UNSET
        if not isinstance(self.on_transform_error, Unset):
            on_transform_error = self.on_transform_error.value

        rate_limit_rps = self.rate_limit_rps

        requires_transform = self.requires_transform

        secret_headers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.secret_headers, Unset):
            secret_headers = self.secret_headers.to_dict()

        signature_type: str | Unset = UNSET
        if not isinstance(self.signature_type, Unset):
            signature_type = self.signature_type.value

        template_missing_key: str | Unset = UNSET
        if not isinstance(self.template_missing_key, Unset):
            template_missing_key = self.template_missing_key.value

        template_notes = self.template_notes

        template_source: str | Unset = UNSET
        if not isinstance(self.template_source, Unset):
            template_source = self.template_source.value

        transform_template = self.transform_template

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "url": url,
            }
        )
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
        if on_transform_error is not UNSET:
            field_dict["on_transform_error"] = on_transform_error
        if rate_limit_rps is not UNSET:
            field_dict["rate_limit_rps"] = rate_limit_rps
        if requires_transform is not UNSET:
            field_dict["requires_transform"] = requires_transform
        if secret_headers is not UNSET:
            field_dict["secret_headers"] = secret_headers
        if signature_type is not UNSET:
            field_dict["signature_type"] = signature_type
        if template_missing_key is not UNSET:
            field_dict["template_missing_key"] = template_missing_key
        if template_notes is not UNSET:
            field_dict["template_notes"] = template_notes
        if template_source is not UNSET:
            field_dict["template_source"] = template_source
        if transform_template is not UNSET:
            field_dict["transform_template"] = transform_template

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.register_webhook_body_headers import RegisterWebhookBodyHeaders  # noqa: PLC0415
        from ..models.register_webhook_body_secret_headers import RegisterWebhookBodySecretHeaders  # noqa: PLC0415
        from ..models.webhook_http_config import WebhookHTTPConfig  # noqa: PLC0415

        d = dict(src_dict)
        url = d.pop("url")

        schema = d.pop("$schema", UNSET)

        active = d.pop("active", UNSET)

        description = d.pop("description", UNSET)

        events = cast(list[str], d.pop("events", UNSET))

        _headers = d.pop("headers", UNSET)
        headers: RegisterWebhookBodyHeaders | Unset
        if isinstance(_headers, Unset):
            headers = UNSET
        else:
            headers = RegisterWebhookBodyHeaders.from_dict(_headers)

        _http_config = d.pop("http_config", UNSET)
        http_config: WebhookHTTPConfig | Unset
        if isinstance(_http_config, Unset):
            http_config = UNSET
        else:
            http_config = WebhookHTTPConfig.from_dict(_http_config)

        _on_transform_error = d.pop("on_transform_error", UNSET)
        on_transform_error: RegisterWebhookBodyOnTransformError | Unset
        if isinstance(_on_transform_error, Unset):
            on_transform_error = UNSET
        else:
            on_transform_error = RegisterWebhookBodyOnTransformError(_on_transform_error)

        rate_limit_rps = d.pop("rate_limit_rps", UNSET)

        requires_transform = d.pop("requires_transform", UNSET)

        _secret_headers = d.pop("secret_headers", UNSET)
        secret_headers: RegisterWebhookBodySecretHeaders | Unset
        if isinstance(_secret_headers, Unset):
            secret_headers = UNSET
        else:
            secret_headers = RegisterWebhookBodySecretHeaders.from_dict(_secret_headers)

        _signature_type = d.pop("signature_type", UNSET)
        signature_type: RegisterWebhookBodySignatureType | Unset
        if isinstance(_signature_type, Unset):
            signature_type = UNSET
        else:
            signature_type = RegisterWebhookBodySignatureType(_signature_type)

        _template_missing_key = d.pop("template_missing_key", UNSET)
        template_missing_key: RegisterWebhookBodyTemplateMissingKey | Unset
        if isinstance(_template_missing_key, Unset):
            template_missing_key = UNSET
        else:
            template_missing_key = RegisterWebhookBodyTemplateMissingKey(_template_missing_key)

        template_notes = d.pop("template_notes", UNSET)

        _template_source = d.pop("template_source", UNSET)
        template_source: RegisterWebhookBodyTemplateSource | Unset
        if isinstance(_template_source, Unset):
            template_source = UNSET
        else:
            template_source = RegisterWebhookBodyTemplateSource(_template_source)

        transform_template = d.pop("transform_template", UNSET)

        register_webhook_body = cls(
            url=url,
            schema=schema,
            active=active,
            description=description,
            events=events,
            headers=headers,
            http_config=http_config,
            on_transform_error=on_transform_error,
            rate_limit_rps=rate_limit_rps,
            requires_transform=requires_transform,
            secret_headers=secret_headers,
            signature_type=signature_type,
            template_missing_key=template_missing_key,
            template_notes=template_notes,
            template_source=template_source,
            transform_template=transform_template,
        )

        return register_webhook_body
