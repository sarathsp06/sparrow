from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.resume_webhook_output_body import ResumeWebhookOutputBody
from ...types import Response


def _get_kwargs(
    consumer: str,
    webhook_id: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/consumers/{consumer}/webhooks/{webhook_id}:resume".format(
            consumer=quote(str(consumer), safe=""),
            webhook_id=quote(str(webhook_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ResumeWebhookOutputBody | None:
    if response.status_code == 200:
        response_200 = ResumeWebhookOutputBody.from_dict(response.json())

        return response_200

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
) -> Response[ErrorModel | ResumeWebhookOutputBody]:
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
) -> Response[ErrorModel | ResumeWebhookOutputBody]:
    """Resume a paused webhook

     Re-enables deliveries to a webhook paused by an operator or auto-disabled, and clears
    auto_disabled_at/auto_disabled_reason. Deliveries held while it was paused stay paused and are never
    sent automatically: the response says how many there are, so you can retry them with
    status=paused&webhook_id=... and the delivery retry endpoints. 409 when the webhook is already
    active.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:resume.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ResumeWebhookOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
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
) -> ErrorModel | ResumeWebhookOutputBody | None:
    """Resume a paused webhook

     Re-enables deliveries to a webhook paused by an operator or auto-disabled, and clears
    auto_disabled_at/auto_disabled_reason. Deliveries held while it was paused stay paused and are never
    sent automatically: the response says how many there are, so you can retry them with
    status=paused&webhook_id=... and the delivery retry endpoints. 409 when the webhook is already
    active.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:resume.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ResumeWebhookOutputBody
    """

    return sync_detailed(
        consumer=consumer,
        webhook_id=webhook_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    webhook_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorModel | ResumeWebhookOutputBody]:
    """Resume a paused webhook

     Re-enables deliveries to a webhook paused by an operator or auto-disabled, and clears
    auto_disabled_at/auto_disabled_reason. Deliveries held while it was paused stay paused and are never
    sent automatically: the response says how many there are, so you can retry them with
    status=paused&webhook_id=... and the delivery retry endpoints. 409 when the webhook is already
    active.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:resume.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ResumeWebhookOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    webhook_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorModel | ResumeWebhookOutputBody | None:
    """Resume a paused webhook

     Re-enables deliveries to a webhook paused by an operator or auto-disabled, and clears
    auto_disabled_at/auto_disabled_reason. Deliveries held while it was paused stay paused and are never
    sent automatically: the response says how many there are, so you can retry them with
    status=paused&webhook_id=... and the delivery retry endpoints. 409 when the webhook is already
    active.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:resume.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ResumeWebhookOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            webhook_id=webhook_id,
            client=client,
        )
    ).parsed
