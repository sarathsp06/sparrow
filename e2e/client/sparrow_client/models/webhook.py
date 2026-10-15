from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.webhook_headers import WebhookHeaders
    from ..models.webhook_secret_headers import WebhookSecretHeaders


T = TypeVar("T", bound="Webhook")


@_attrs_define
class Webhook:
    """
    Attributes:
        url (str): Destination URL; may contain {{param}} tokens.
        headers (WebhookHeaders | Unset): Static headers sent with every delivery; values may contain {{param}} tokens.
        requires_transform (bool | Unset): The destination only accepts the transformed payload; the webhook is
            registered with requires_transform.
        secret_headers (WebhookSecretHeaders | Unset): Headers whose values are encrypted at rest and masked in API
            responses (upstream tokens, passwords); values may contain {{param}} tokens.
    """

    url: str
    headers: WebhookHeaders | Unset = UNSET
    requires_transform: bool | Unset = UNSET
    secret_headers: WebhookSecretHeaders | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        url = self.url

        headers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.headers, Unset):
            headers = self.headers.to_dict()

        requires_transform = self.requires_transform

        secret_headers: dict[str, Any] | Unset = UNSET
        if not isinstance(self.secret_headers, Unset):
            secret_headers = self.secret_headers.to_dict()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "url": url,
            }
        )
        if headers is not UNSET:
            field_dict["headers"] = headers
        if requires_transform is not UNSET:
            field_dict["requires_transform"] = requires_transform
        if secret_headers is not UNSET:
            field_dict["secret_headers"] = secret_headers

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.webhook_headers import WebhookHeaders  # noqa: PLC0415
        from ..models.webhook_secret_headers import WebhookSecretHeaders  # noqa: PLC0415

        d = dict(src_dict)
        url = d.pop("url")

        _headers = d.pop("headers", UNSET)
        headers: WebhookHeaders | Unset
        if isinstance(_headers, Unset):
            headers = UNSET
        else:
            headers = WebhookHeaders.from_dict(_headers)

        requires_transform = d.pop("requires_transform", UNSET)

        _secret_headers = d.pop("secret_headers", UNSET)
        secret_headers: WebhookSecretHeaders | Unset
        if isinstance(_secret_headers, Unset):
            secret_headers = UNSET
        else:
            secret_headers = WebhookSecretHeaders.from_dict(_secret_headers)

        webhook = cls(
            url=url,
            headers=headers,
            requires_transform=requires_transform,
            secret_headers=secret_headers,
        )

        return webhook
