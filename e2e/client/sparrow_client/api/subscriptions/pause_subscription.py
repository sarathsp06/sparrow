from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.pause_subscription_input_body import PauseSubscriptionInputBody
from ...models.subscription_item import SubscriptionItem
from ...types import Response


def _get_kwargs(
    consumer: str,
    subscription_id: str,
    *,
    body: PauseSubscriptionInputBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/consumers/{consumer}/subscriptions/{subscription_id}:pause".format(
            consumer=quote(str(consumer), safe=""),
            subscription_id=quote(str(subscription_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | SubscriptionItem | None:
    if response.status_code == 200:
        response_200 = SubscriptionItem.from_dict(response.json())

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
    body: PauseSubscriptionInputBody,
) -> Response[ErrorModel | SubscriptionItem]:
    """Pause a subscription

     Stops the subscription's deliveries from being attempted, for any reason (receiver maintenance, a
    template being fixed, an investigation). Events keep fanning out to it: each delivery is recorded
    with status paused and nothing is queued, so nothing is lost. Queued or retrying deliveries of the
    subscription are held as paused when their turn comes. A pause never affects the webhook's health.
    Returns the subscription (200), unlike webhook pause, which returns 204.

    Idempotent: pausing again updates the reason and keeps the original pause time. 400 for a reason
    over 500 characters, 404 when it is not in this consumer.

    Portal tokens: allowed, as POST /portal/api/subscriptions/{subscription_id}:pause.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).
        body (PauseSubscriptionInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | SubscriptionItem]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        subscription_id=subscription_id,
        body=body,
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
    body: PauseSubscriptionInputBody,
) -> ErrorModel | SubscriptionItem | None:
    """Pause a subscription

     Stops the subscription's deliveries from being attempted, for any reason (receiver maintenance, a
    template being fixed, an investigation). Events keep fanning out to it: each delivery is recorded
    with status paused and nothing is queued, so nothing is lost. Queued or retrying deliveries of the
    subscription are held as paused when their turn comes. A pause never affects the webhook's health.
    Returns the subscription (200), unlike webhook pause, which returns 204.

    Idempotent: pausing again updates the reason and keeps the original pause time. 400 for a reason
    over 500 characters, 404 when it is not in this consumer.

    Portal tokens: allowed, as POST /portal/api/subscriptions/{subscription_id}:pause.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).
        body (PauseSubscriptionInputBody):

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
        body=body,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    subscription_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: PauseSubscriptionInputBody,
) -> Response[ErrorModel | SubscriptionItem]:
    """Pause a subscription

     Stops the subscription's deliveries from being attempted, for any reason (receiver maintenance, a
    template being fixed, an investigation). Events keep fanning out to it: each delivery is recorded
    with status paused and nothing is queued, so nothing is lost. Queued or retrying deliveries of the
    subscription are held as paused when their turn comes. A pause never affects the webhook's health.
    Returns the subscription (200), unlike webhook pause, which returns 204.

    Idempotent: pausing again updates the reason and keeps the original pause time. 400 for a reason
    over 500 characters, 404 when it is not in this consumer.

    Portal tokens: allowed, as POST /portal/api/subscriptions/{subscription_id}:pause.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).
        body (PauseSubscriptionInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | SubscriptionItem]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        subscription_id=subscription_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    subscription_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: PauseSubscriptionInputBody,
) -> ErrorModel | SubscriptionItem | None:
    """Pause a subscription

     Stops the subscription's deliveries from being attempted, for any reason (receiver maintenance, a
    template being fixed, an investigation). Events keep fanning out to it: each delivery is recorded
    with status paused and nothing is queued, so nothing is lost. Queued or retrying deliveries of the
    subscription are held as paused when their turn comes. A pause never affects the webhook's health.
    Returns the subscription (200), unlike webhook pause, which returns 204.

    Idempotent: pausing again updates the reason and keeps the original pause time. 400 for a reason
    over 500 characters, 404 when it is not in this consumer.

    Portal tokens: allowed, as POST /portal/api/subscriptions/{subscription_id}:pause.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).
        body (PauseSubscriptionInputBody):

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
            body=body,
        )
    ).parsed
