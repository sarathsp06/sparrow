from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.retry_output_body import RetryOutputBody
from ...types import Response


def _get_kwargs(
    consumer: str,
    delivery_id: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/consumers/{consumer}/deliveries/{delivery_id}:retry".format(
            consumer=quote(str(consumer), safe=""),
            delivery_id=quote(str(delivery_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | RetryOutputBody | None:
    if response.status_code == 200:
        response_200 = RetryOutputBody.from_dict(response.json())

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
) -> Response[ErrorModel | RetryOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    consumer: str,
    delivery_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorModel | RetryOutputBody]:
    """Retry a single delivery

     Queues one delivery for another round of attempts right away: it is reset to pending and gets up to
    max_attempts new attempts, without TTL expiry. Works for failed, expired, paused, pending and
    retrying deliveries; a successful one is refused with 409 (use the bulk route with force to resend
    it). If the webhook or subscription is still paused the delivery is held as paused again. Not
    idempotent: each call queues another round.

    Portal tokens: allowed, as POST /portal/api/deliveries/{delivery_id}:retry.

    Args:
        consumer (str): Consumer the delivery belongs to.
        delivery_id (str): Delivery id (UUID) to retry.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | RetryOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        delivery_id=delivery_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    delivery_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorModel | RetryOutputBody | None:
    """Retry a single delivery

     Queues one delivery for another round of attempts right away: it is reset to pending and gets up to
    max_attempts new attempts, without TTL expiry. Works for failed, expired, paused, pending and
    retrying deliveries; a successful one is refused with 409 (use the bulk route with force to resend
    it). If the webhook or subscription is still paused the delivery is held as paused again. Not
    idempotent: each call queues another round.

    Portal tokens: allowed, as POST /portal/api/deliveries/{delivery_id}:retry.

    Args:
        consumer (str): Consumer the delivery belongs to.
        delivery_id (str): Delivery id (UUID) to retry.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | RetryOutputBody
    """

    return sync_detailed(
        consumer=consumer,
        delivery_id=delivery_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    delivery_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorModel | RetryOutputBody]:
    """Retry a single delivery

     Queues one delivery for another round of attempts right away: it is reset to pending and gets up to
    max_attempts new attempts, without TTL expiry. Works for failed, expired, paused, pending and
    retrying deliveries; a successful one is refused with 409 (use the bulk route with force to resend
    it). If the webhook or subscription is still paused the delivery is held as paused again. Not
    idempotent: each call queues another round.

    Portal tokens: allowed, as POST /portal/api/deliveries/{delivery_id}:retry.

    Args:
        consumer (str): Consumer the delivery belongs to.
        delivery_id (str): Delivery id (UUID) to retry.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | RetryOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        delivery_id=delivery_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    delivery_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorModel | RetryOutputBody | None:
    """Retry a single delivery

     Queues one delivery for another round of attempts right away: it is reset to pending and gets up to
    max_attempts new attempts, without TTL expiry. Works for failed, expired, paused, pending and
    retrying deliveries; a successful one is refused with 409 (use the bulk route with force to resend
    it). If the webhook or subscription is still paused the delivery is held as paused again. Not
    idempotent: each call queues another round.

    Portal tokens: allowed, as POST /portal/api/deliveries/{delivery_id}:retry.

    Args:
        consumer (str): Consumer the delivery belongs to.
        delivery_id (str): Delivery id (UUID) to retry.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | RetryOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            delivery_id=delivery_id,
            client=client,
        )
    ).parsed
