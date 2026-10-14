from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.create_subscription_body import CreateSubscriptionBody
from ...models.error_model import ErrorModel
from ...models.subscription_item import SubscriptionItem
from ...types import Response


def _get_kwargs(
    consumer: str,
    *,
    body: CreateSubscriptionBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/consumers/{consumer}/subscriptions".format(
            consumer=quote(str(consumer), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | SubscriptionItem | None:
    if response.status_code == 201:
        response_201 = SubscriptionItem.from_dict(response.json())

        return response_201

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
) -> Response[ErrorModel | SubscriptionItem]:
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
    body: CreateSubscriptionBody,
) -> Response[ErrorModel | SubscriptionItem]:
    """Create a subscription linking a webhook to an event

     Subscribes a webhook to an event type within a consumer, with an optional payload transform and
    label filters. Registering a webhook already auto-creates one subscription per listed event — use
    this endpoint for additional or catch-all ("*") subscriptions.

    Args:
        consumer (str):
        body (CreateSubscriptionBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | SubscriptionItem]
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
    body: CreateSubscriptionBody,
) -> ErrorModel | SubscriptionItem | None:
    """Create a subscription linking a webhook to an event

     Subscribes a webhook to an event type within a consumer, with an optional payload transform and
    label filters. Registering a webhook already auto-creates one subscription per listed event — use
    this endpoint for additional or catch-all ("*") subscriptions.

    Args:
        consumer (str):
        body (CreateSubscriptionBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | SubscriptionItem
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
    body: CreateSubscriptionBody,
) -> Response[ErrorModel | SubscriptionItem]:
    """Create a subscription linking a webhook to an event

     Subscribes a webhook to an event type within a consumer, with an optional payload transform and
    label filters. Registering a webhook already auto-creates one subscription per listed event — use
    this endpoint for additional or catch-all ("*") subscriptions.

    Args:
        consumer (str):
        body (CreateSubscriptionBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | SubscriptionItem]
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
    body: CreateSubscriptionBody,
) -> ErrorModel | SubscriptionItem | None:
    """Create a subscription linking a webhook to an event

     Subscribes a webhook to an event type within a consumer, with an optional payload transform and
    label filters. Registering a webhook already auto-creates one subscription per listed event — use
    this endpoint for additional or catch-all ("*") subscriptions.

    Args:
        consumer (str):
        body (CreateSubscriptionBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | SubscriptionItem
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            client=client,
            body=body,
        )
    ).parsed
