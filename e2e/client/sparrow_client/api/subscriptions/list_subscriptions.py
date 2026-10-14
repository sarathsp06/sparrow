from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_subscriptions_output_body import ListSubscriptionsOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    consumer: str,
    *,
    webhook_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["webhook_id"] = webhook_id

    params["event_name"] = event_name

    params["limit"] = limit

    params["offset"] = offset

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/consumers/{consumer}/subscriptions".format(
            consumer=quote(str(consumer), safe=""),
        ),
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
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListSubscriptionsOutputBody]:
    """List subscriptions

     Lists subscriptions in a consumer, optionally filtered by webhook or event type name.

    Args:
        consumer (str): Tenant consumer to list subscriptions in.
        webhook_id (str | Unset): Filter to subscriptions for one webhook.
        event_name (str | Unset): Filter to subscriptions for one event type name.
        limit (int | Unset): Maximum items to return. Default: 50.
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
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> ErrorModel | ListSubscriptionsOutputBody | None:
    """List subscriptions

     Lists subscriptions in a consumer, optionally filtered by webhook or event type name.

    Args:
        consumer (str): Tenant consumer to list subscriptions in.
        webhook_id (str | Unset): Filter to subscriptions for one webhook.
        event_name (str | Unset): Filter to subscriptions for one event type name.
        limit (int | Unset): Maximum items to return. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListSubscriptionsOutputBody
    """

    return sync_detailed(
        consumer=consumer,
        client=client,
        webhook_id=webhook_id,
        event_name=event_name,
        limit=limit,
        offset=offset,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListSubscriptionsOutputBody]:
    """List subscriptions

     Lists subscriptions in a consumer, optionally filtered by webhook or event type name.

    Args:
        consumer (str): Tenant consumer to list subscriptions in.
        webhook_id (str | Unset): Filter to subscriptions for one webhook.
        event_name (str | Unset): Filter to subscriptions for one event type name.
        limit (int | Unset): Maximum items to return. Default: 50.
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
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> ErrorModel | ListSubscriptionsOutputBody | None:
    """List subscriptions

     Lists subscriptions in a consumer, optionally filtered by webhook or event type name.

    Args:
        consumer (str): Tenant consumer to list subscriptions in.
        webhook_id (str | Unset): Filter to subscriptions for one webhook.
        event_name (str | Unset): Filter to subscriptions for one event type name.
        limit (int | Unset): Maximum items to return. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListSubscriptionsOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            client=client,
            webhook_id=webhook_id,
            event_name=event_name,
            limit=limit,
            offset=offset,
        )
    ).parsed
