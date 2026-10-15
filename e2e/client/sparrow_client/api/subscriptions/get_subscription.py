from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.subscription_item import SubscriptionItem
from ...types import Response


def _get_kwargs(
    consumer: str,
    subscription_id: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/consumers/{consumer}/subscriptions/{subscription_id}".format(
            consumer=quote(str(consumer), safe=""),
            subscription_id=quote(str(subscription_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | SubscriptionItem | None:
    if response.status_code == 200:
        response_200 = SubscriptionItem.from_dict(response.json())

        return response_200

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
) -> Response[ErrorModel | SubscriptionItem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    consumer: str,
    subscription_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorModel | SubscriptionItem]:
    """Get a subscription by id

     Returns one subscription: its webhook link, transform template and settings, label filters and pause
    state. 404 when it is not in this consumer.

    Portal tokens: allowed, as GET /portal/api/subscriptions/{subscription_id}.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | SubscriptionItem]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        subscription_id=subscription_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    subscription_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorModel | SubscriptionItem | None:
    """Get a subscription by id

     Returns one subscription: its webhook link, transform template and settings, label filters and pause
    state. 404 when it is not in this consumer.

    Portal tokens: allowed, as GET /portal/api/subscriptions/{subscription_id}.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | SubscriptionItem
    """

    return sync_detailed(
        consumer=consumer,
        subscription_id=subscription_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    subscription_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorModel | SubscriptionItem]:
    """Get a subscription by id

     Returns one subscription: its webhook link, transform template and settings, label filters and pause
    state. 404 when it is not in this consumer.

    Portal tokens: allowed, as GET /portal/api/subscriptions/{subscription_id}.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | SubscriptionItem]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        subscription_id=subscription_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    subscription_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorModel | SubscriptionItem | None:
    """Get a subscription by id

     Returns one subscription: its webhook link, transform template and settings, label filters and pause
    state. 404 when it is not in this consumer.

    Portal tokens: allowed, as GET /portal/api/subscriptions/{subscription_id}.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | SubscriptionItem
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            subscription_id=subscription_id,
            client=client,
        )
    ).parsed
