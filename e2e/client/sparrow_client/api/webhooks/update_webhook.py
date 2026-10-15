from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.patch_webhook_body import PatchWebhookBody
from ...models.webhook_out import WebhookOut
from ...types import Response


def _get_kwargs(
    consumer: str,
    webhook_id: str,
    *,
    body: PatchWebhookBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/v1/consumers/{consumer}/webhooks/{webhook_id}".format(
            consumer=quote(str(consumer), safe=""),
            webhook_id=quote(str(webhook_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | WebhookOut | None:
    if response.status_code == 200:
        response_200 = WebhookOut.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = ErrorModel.from_dict(response.json())

        return response_400

    if response.status_code == 404:
        response_404 = ErrorModel.from_dict(response.json())

        return response_404

    if response.status_code == 409:
        response_409 = ErrorModel.from_dict(response.json())

        return response_409

    if response.status_code == 422:
        response_422 = ErrorModel.from_dict(response.json())

        return response_422

    if response.status_code == 500:
        response_500 = ErrorModel.from_dict(response.json())

        return response_500

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorModel | WebhookOut]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    consumer: str,
    webhook_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: PatchWebhookBody,
) -> Response[ErrorModel | WebhookOut]:
    """Partially update a webhook

     Partially updates a webhook: only fields present in the body change. Within http_config only the
    sent fields change, and the merged result is validated as on register (400 if out of range). Sending
    events replaces every subscription of the webhook (see events). Setting active toggles delivery like
    :pause/:resume but never fails when the state is already as requested. Returns the updated webhook.
    Idempotent: repeating the same body leaves the same state.

    409 when turning requires_transform on while a subscription has no enabled transform, or replacing
    events while it is on. The deprecated top-level timeout is applied only when
    http_config.request_timeout_seconds is not sent.

    Portal tokens: allowed, as PATCH /portal/api/webhooks/{webhook_id}.

    Args:
        consumer (str): Consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (PatchWebhookBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | WebhookOut]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    webhook_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: PatchWebhookBody,
) -> ErrorModel | WebhookOut | None:
    """Partially update a webhook

     Partially updates a webhook: only fields present in the body change. Within http_config only the
    sent fields change, and the merged result is validated as on register (400 if out of range). Sending
    events replaces every subscription of the webhook (see events). Setting active toggles delivery like
    :pause/:resume but never fails when the state is already as requested. Returns the updated webhook.
    Idempotent: repeating the same body leaves the same state.

    409 when turning requires_transform on while a subscription has no enabled transform, or replacing
    events while it is on. The deprecated top-level timeout is applied only when
    http_config.request_timeout_seconds is not sent.

    Portal tokens: allowed, as PATCH /portal/api/webhooks/{webhook_id}.

    Args:
        consumer (str): Consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (PatchWebhookBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | WebhookOut
    """

    return sync_detailed(
        consumer=consumer,
        webhook_id=webhook_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    webhook_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: PatchWebhookBody,
) -> Response[ErrorModel | WebhookOut]:
    """Partially update a webhook

     Partially updates a webhook: only fields present in the body change. Within http_config only the
    sent fields change, and the merged result is validated as on register (400 if out of range). Sending
    events replaces every subscription of the webhook (see events). Setting active toggles delivery like
    :pause/:resume but never fails when the state is already as requested. Returns the updated webhook.
    Idempotent: repeating the same body leaves the same state.

    409 when turning requires_transform on while a subscription has no enabled transform, or replacing
    events while it is on. The deprecated top-level timeout is applied only when
    http_config.request_timeout_seconds is not sent.

    Portal tokens: allowed, as PATCH /portal/api/webhooks/{webhook_id}.

    Args:
        consumer (str): Consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (PatchWebhookBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | WebhookOut]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    webhook_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: PatchWebhookBody,
) -> ErrorModel | WebhookOut | None:
    """Partially update a webhook

     Partially updates a webhook: only fields present in the body change. Within http_config only the
    sent fields change, and the merged result is validated as on register (400 if out of range). Sending
    events replaces every subscription of the webhook (see events). Setting active toggles delivery like
    :pause/:resume but never fails when the state is already as requested. Returns the updated webhook.
    Idempotent: repeating the same body leaves the same state.

    409 when turning requires_transform on while a subscription has no enabled transform, or replacing
    events while it is on. The deprecated top-level timeout is applied only when
    http_config.request_timeout_seconds is not sent.

    Portal tokens: allowed, as PATCH /portal/api/webhooks/{webhook_id}.

    Args:
        consumer (str): Consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (PatchWebhookBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | WebhookOut
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            webhook_id=webhook_id,
            client=client,
            body=body,
        )
    ).parsed
