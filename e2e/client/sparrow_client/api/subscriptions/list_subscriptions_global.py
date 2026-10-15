from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_subscriptions_output_body import ListSubscriptionsOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    consumer: str | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["consumer"] = consumer

    params["webhook_id"] = webhook_id

    params["event_name"] = event_name

    params["limit"] = limit

    params["offset"] = offset

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/subscriptions",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ListSubscriptionsOutputBody:
    if response.status_code == 200:
        response_200 = ListSubscriptionsOutputBody.from_dict(response.json())

        return response_200

    response_default = ErrorModel.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorModel | ListSubscriptionsOutputBody]:
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
    webhook_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListSubscriptionsOutputBody]:
    """List subscriptions across all consumers

     Lists subscriptions across every consumer with the same filters and pagination as the per-consumer
    route. Pass consumer to scope to one consumer.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list subscriptions across all
            consumers.
        webhook_id (str | Unset): Filter to subscriptions for one webhook (UUID). Takes precedence
            over event_name: when both are given, event_name is ignored.
        event_name (str | Unset): Filter to subscriptions with exactly this event_name ("*"
            matches only catch-all subscriptions).
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListSubscriptionsOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
        event_name=event_name,
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
    webhook_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> ErrorModel | ListSubscriptionsOutputBody | None:
    """List subscriptions across all consumers

     Lists subscriptions across every consumer with the same filters and pagination as the per-consumer
    route. Pass consumer to scope to one consumer.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list subscriptions across all
            consumers.
        webhook_id (str | Unset): Filter to subscriptions for one webhook (UUID). Takes precedence
            over event_name: when both are given, event_name is ignored.
        event_name (str | Unset): Filter to subscriptions with exactly this event_name ("*"
            matches only catch-all subscriptions).
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListSubscriptionsOutputBody
    """

    return sync_detailed(
        client=client,
        consumer=consumer,
        webhook_id=webhook_id,
        event_name=event_name,
        limit=limit,
        offset=offset,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListSubscriptionsOutputBody]:
    """List subscriptions across all consumers

     Lists subscriptions across every consumer with the same filters and pagination as the per-consumer
    route. Pass consumer to scope to one consumer.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list subscriptions across all
            consumers.
        webhook_id (str | Unset): Filter to subscriptions for one webhook (UUID). Takes precedence
            over event_name: when both are given, event_name is ignored.
        event_name (str | Unset): Filter to subscriptions with exactly this event_name ("*"
            matches only catch-all subscriptions).
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListSubscriptionsOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
        event_name=event_name,
        limit=limit,
        offset=offset,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> ErrorModel | ListSubscriptionsOutputBody | None:
    """List subscriptions across all consumers

     Lists subscriptions across every consumer with the same filters and pagination as the per-consumer
    route. Pass consumer to scope to one consumer.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list subscriptions across all
            consumers.
        webhook_id (str | Unset): Filter to subscriptions for one webhook (UUID). Takes precedence
            over event_name: when both are given, event_name is ignored.
        event_name (str | Unset): Filter to subscriptions with exactly this event_name ("*"
            matches only catch-all subscriptions).
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListSubscriptionsOutputBody
    """

    return (
        await asyncio_detailed(
            client=client,
            consumer=consumer,
            webhook_id=webhook_id,
            event_name=event_name,
            limit=limit,
            offset=offset,
        )
    ).parsed
