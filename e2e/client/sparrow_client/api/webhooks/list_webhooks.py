from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_webhooks_health import ListWebhooksHealth
from ...models.list_webhooks_output_body import ListWebhooksOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    consumer: str,
    *,
    webhook_id: str | Unset = UNSET,
    event: str | Unset = UNSET,
    active: bool | Unset = False,
    health: ListWebhooksHealth | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["webhook_id"] = webhook_id

    params["event"] = event

    params["active"] = active

    json_health: str | Unset = UNSET
    if not isinstance(health, Unset):
        json_health = health.value

    params["health"] = json_health

    params["limit"] = limit

    params["offset"] = offset

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/consumers/{consumer}/webhooks".format(
            consumer=quote(str(consumer), safe=""),
        ),
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
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
    event: str | Unset = UNSET,
    active: bool | Unset = False,
    health: ListWebhooksHealth | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListWebhooksOutputBody]:
    """List webhooks

     Lists the webhooks of one consumer, newest first, optionally filtered by id, subscribed event,
    active flag, or health status. Secrets are masked. Offset paginated with total_count. Deleted
    webhooks are never listed.

    Portal tokens: allowed, as GET /portal/api/webhooks.

    Args:
        consumer (str): Tenant consumer to list webhooks in.
        webhook_id (str | Unset): Filter to a single webhook by id.
        event (str | Unset): Filter to webhooks subscribed to this event type name.
        active (bool | Unset): true returns only active (not paused) webhooks; false returns all.
            Default: False.
        health (ListWebhooksHealth | Unset): Filter by computed health status.
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListWebhooksOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
        event=event,
        active=active,
        health=health,
        limit=limit,
        offset=offset,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
    event: str | Unset = UNSET,
    active: bool | Unset = False,
    health: ListWebhooksHealth | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> ErrorModel | ListWebhooksOutputBody | None:
    """List webhooks

     Lists the webhooks of one consumer, newest first, optionally filtered by id, subscribed event,
    active flag, or health status. Secrets are masked. Offset paginated with total_count. Deleted
    webhooks are never listed.

    Portal tokens: allowed, as GET /portal/api/webhooks.

    Args:
        consumer (str): Tenant consumer to list webhooks in.
        webhook_id (str | Unset): Filter to a single webhook by id.
        event (str | Unset): Filter to webhooks subscribed to this event type name.
        active (bool | Unset): true returns only active (not paused) webhooks; false returns all.
            Default: False.
        health (ListWebhooksHealth | Unset): Filter by computed health status.
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListWebhooksOutputBody
    """

    return sync_detailed(
        consumer=consumer,
        client=client,
        webhook_id=webhook_id,
        event=event,
        active=active,
        health=health,
        limit=limit,
        offset=offset,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
    event: str | Unset = UNSET,
    active: bool | Unset = False,
    health: ListWebhooksHealth | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListWebhooksOutputBody]:
    """List webhooks

     Lists the webhooks of one consumer, newest first, optionally filtered by id, subscribed event,
    active flag, or health status. Secrets are masked. Offset paginated with total_count. Deleted
    webhooks are never listed.

    Portal tokens: allowed, as GET /portal/api/webhooks.

    Args:
        consumer (str): Tenant consumer to list webhooks in.
        webhook_id (str | Unset): Filter to a single webhook by id.
        event (str | Unset): Filter to webhooks subscribed to this event type name.
        active (bool | Unset): true returns only active (not paused) webhooks; false returns all.
            Default: False.
        health (ListWebhooksHealth | Unset): Filter by computed health status.
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListWebhooksOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
        event=event,
        active=active,
        health=health,
        limit=limit,
        offset=offset,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
    event: str | Unset = UNSET,
    active: bool | Unset = False,
    health: ListWebhooksHealth | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> ErrorModel | ListWebhooksOutputBody | None:
    """List webhooks

     Lists the webhooks of one consumer, newest first, optionally filtered by id, subscribed event,
    active flag, or health status. Secrets are masked. Offset paginated with total_count. Deleted
    webhooks are never listed.

    Portal tokens: allowed, as GET /portal/api/webhooks.

    Args:
        consumer (str): Tenant consumer to list webhooks in.
        webhook_id (str | Unset): Filter to a single webhook by id.
        event (str | Unset): Filter to webhooks subscribed to this event type name.
        active (bool | Unset): true returns only active (not paused) webhooks; false returns all.
            Default: False.
        health (ListWebhooksHealth | Unset): Filter by computed health status.
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListWebhooksOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            client=client,
            webhook_id=webhook_id,
            event=event,
            active=active,
            health=health,
            limit=limit,
            offset=offset,
        )
    ).parsed
