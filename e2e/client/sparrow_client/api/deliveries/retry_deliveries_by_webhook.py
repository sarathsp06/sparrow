from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.retry_deliveries_by_webhook_input_body import RetryDeliveriesByWebhookInputBody
from ...models.retry_output_body import RetryOutputBody
from ...types import Response


def _get_kwargs(
    consumer: str,
    *,
    body: RetryDeliveriesByWebhookInputBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/consumers/{consumer}/deliveries:retry".format(
            consumer=quote(str(consumer), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
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
    *,
    client: AuthenticatedClient | Client,
    body: RetryDeliveriesByWebhookInputBody,
) -> Response[ErrorModel | RetryOutputBody]:
    """Retry deliveries in bulk for a webhook

     Queues every failed, pending, retrying or paused delivery of one webhook for another round of
    attempts (see retryDelivery), in one request. With force, every delivery of the webhook is resent,
    including successful and expired ones. Returns count 0 when nothing matched. For a filtered or very
    large set, use prepare_retry and :retryBatch instead.

    404 when the webhook does not exist in this consumer.

    Portal tokens: allowed, as POST /portal/api/deliveries:retry, unless the server runs with
    SPARROW_PORTAL_RETRY=false (then 403).

    Args:
        consumer (str): Consumer the webhook belongs to.
        body (RetryDeliveriesByWebhookInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | RetryOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    body: RetryDeliveriesByWebhookInputBody,
) -> ErrorModel | RetryOutputBody | None:
    """Retry deliveries in bulk for a webhook

     Queues every failed, pending, retrying or paused delivery of one webhook for another round of
    attempts (see retryDelivery), in one request. With force, every delivery of the webhook is resent,
    including successful and expired ones. Returns count 0 when nothing matched. For a filtered or very
    large set, use prepare_retry and :retryBatch instead.

    404 when the webhook does not exist in this consumer.

    Portal tokens: allowed, as POST /portal/api/deliveries:retry, unless the server runs with
    SPARROW_PORTAL_RETRY=false (then 403).

    Args:
        consumer (str): Consumer the webhook belongs to.
        body (RetryDeliveriesByWebhookInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | RetryOutputBody
    """

    return sync_detailed(
        consumer=consumer,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    body: RetryDeliveriesByWebhookInputBody,
) -> Response[ErrorModel | RetryOutputBody]:
    """Retry deliveries in bulk for a webhook

     Queues every failed, pending, retrying or paused delivery of one webhook for another round of
    attempts (see retryDelivery), in one request. With force, every delivery of the webhook is resent,
    including successful and expired ones. Returns count 0 when nothing matched. For a filtered or very
    large set, use prepare_retry and :retryBatch instead.

    404 when the webhook does not exist in this consumer.

    Portal tokens: allowed, as POST /portal/api/deliveries:retry, unless the server runs with
    SPARROW_PORTAL_RETRY=false (then 403).

    Args:
        consumer (str): Consumer the webhook belongs to.
        body (RetryDeliveriesByWebhookInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | RetryOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    body: RetryDeliveriesByWebhookInputBody,
) -> ErrorModel | RetryOutputBody | None:
    """Retry deliveries in bulk for a webhook

     Queues every failed, pending, retrying or paused delivery of one webhook for another round of
    attempts (see retryDelivery), in one request. With force, every delivery of the webhook is resent,
    including successful and expired ones. Returns count 0 when nothing matched. For a filtered or very
    large set, use prepare_retry and :retryBatch instead.

    404 when the webhook does not exist in this consumer.

    Portal tokens: allowed, as POST /portal/api/deliveries:retry, unless the server runs with
    SPARROW_PORTAL_RETRY=false (then 403).

    Args:
        consumer (str): Consumer the webhook belongs to.
        body (RetryDeliveriesByWebhookInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | RetryOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            client=client,
            body=body,
        )
    ).parsed
