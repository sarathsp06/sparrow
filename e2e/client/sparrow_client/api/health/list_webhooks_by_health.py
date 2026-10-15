from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_webhooks_by_health_health import ListWebhooksByHealthHealth
from ...models.list_webhooks_output_body import ListWebhooksOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    consumer: str | Unset = UNSET,
    health: ListWebhooksByHealthHealth | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["consumer"] = consumer

    json_health: str | Unset = UNSET
    if not isinstance(health, Unset):
        json_health = health.value

    params["health"] = json_health

    params["webhook_id"] = webhook_id

    params["limit"] = limit

    params["offset"] = offset

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/webhooks",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ListWebhooksOutputBody:
    if response.status_code == 200:
        response_200 = ListWebhooksOutputBody.from_dict(response.json())

        return response_200

    response_default = ErrorModel.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorModel | ListWebhooksOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    health: ListWebhooksByHealthHealth | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListWebhooksOutputBody]:
    """List webhooks across all consumers

     Lists webhooks across every consumer, newest first, with masked secrets. Pass consumer to scope to
    one consumer, health to filter by label, or webhook_id for an id-only lookup when the consumer is
    not known. Offset paginated with total_count.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list webhooks across all
            consumers.
        health (ListWebhooksByHealthHealth | Unset): Filter to webhooks with this computed health
            status.
        webhook_id (str | Unset): Look up a single webhook by id when its consumer is unknown.
        limit (int | Unset): Maximum items to return; values above 1000 are capped at 1000.
            Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListWebhooksOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        health=health,
        webhook_id=webhook_id,
        limit=limit,
        offset=offset,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    health: ListWebhooksByHealthHealth | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> ErrorModel | ListWebhooksOutputBody | None:
    """List webhooks across all consumers

     Lists webhooks across every consumer, newest first, with masked secrets. Pass consumer to scope to
    one consumer, health to filter by label, or webhook_id for an id-only lookup when the consumer is
    not known. Offset paginated with total_count.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list webhooks across all
            consumers.
        health (ListWebhooksByHealthHealth | Unset): Filter to webhooks with this computed health
            status.
        webhook_id (str | Unset): Look up a single webhook by id when its consumer is unknown.
        limit (int | Unset): Maximum items to return; values above 1000 are capped at 1000.
            Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListWebhooksOutputBody
    """

    return sync_detailed(
        client=client,
        consumer=consumer,
        health=health,
        webhook_id=webhook_id,
        limit=limit,
        offset=offset,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    health: ListWebhooksByHealthHealth | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListWebhooksOutputBody]:
    """List webhooks across all consumers

     Lists webhooks across every consumer, newest first, with masked secrets. Pass consumer to scope to
    one consumer, health to filter by label, or webhook_id for an id-only lookup when the consumer is
    not known. Offset paginated with total_count.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list webhooks across all
            consumers.
        health (ListWebhooksByHealthHealth | Unset): Filter to webhooks with this computed health
            status.
        webhook_id (str | Unset): Look up a single webhook by id when its consumer is unknown.
        limit (int | Unset): Maximum items to return; values above 1000 are capped at 1000.
            Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListWebhooksOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        health=health,
        webhook_id=webhook_id,
        limit=limit,
        offset=offset,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    health: ListWebhooksByHealthHealth | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> ErrorModel | ListWebhooksOutputBody | None:
    """List webhooks across all consumers

     Lists webhooks across every consumer, newest first, with masked secrets. Pass consumer to scope to
    one consumer, health to filter by label, or webhook_id for an id-only lookup when the consumer is
    not known. Offset paginated with total_count.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list webhooks across all
            consumers.
        health (ListWebhooksByHealthHealth | Unset): Filter to webhooks with this computed health
            status.
        webhook_id (str | Unset): Look up a single webhook by id when its consumer is unknown.
        limit (int | Unset): Maximum items to return; values above 1000 are capped at 1000.
            Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListWebhooksOutputBody
    """

    return (
        await asyncio_detailed(
            client=client,
            consumer=consumer,
            health=health,
            webhook_id=webhook_id,
            limit=limit,
            offset=offset,
        )
    ).parsed
