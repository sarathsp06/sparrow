from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.resume_subscription_output_body import ResumeSubscriptionOutputBody
from ...types import Response


def _get_kwargs(
    consumer: str,
    subscription_id: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/consumers/{consumer}/subscriptions/{subscription_id}:resume".format(
            consumer=quote(str(consumer), safe=""),
            subscription_id=quote(str(subscription_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ResumeSubscriptionOutputBody | None:
    if response.status_code == 200:
        response_200 = ResumeSubscriptionOutputBody.from_dict(response.json())

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
) -> Response[ErrorModel | ResumeSubscriptionOutputBody]:
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
) -> Response[ErrorModel | ResumeSubscriptionOutputBody]:
    """Resume a paused subscription

     Lets new deliveries be attempted again. Deliveries recorded while paused stay paused and are never
    sent automatically: the response says how many there are and when the pause began, so you can retry
    all of them or only those since a time with the delivery retry endpoints. Idempotent: resuming a
    subscription that is not paused succeeds with no paused_since.

    Portal tokens: allowed, as POST /portal/api/subscriptions/{subscription_id}:resume.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ResumeSubscriptionOutputBody]
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
) -> ErrorModel | ResumeSubscriptionOutputBody | None:
    """Resume a paused subscription

     Lets new deliveries be attempted again. Deliveries recorded while paused stay paused and are never
    sent automatically: the response says how many there are and when the pause began, so you can retry
    all of them or only those since a time with the delivery retry endpoints. Idempotent: resuming a
    subscription that is not paused succeeds with no paused_since.

    Portal tokens: allowed, as POST /portal/api/subscriptions/{subscription_id}:resume.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ResumeSubscriptionOutputBody
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
) -> Response[ErrorModel | ResumeSubscriptionOutputBody]:
    """Resume a paused subscription

     Lets new deliveries be attempted again. Deliveries recorded while paused stay paused and are never
    sent automatically: the response says how many there are and when the pause began, so you can retry
    all of them or only those since a time with the delivery retry endpoints. Idempotent: resuming a
    subscription that is not paused succeeds with no paused_since.

    Portal tokens: allowed, as POST /portal/api/subscriptions/{subscription_id}:resume.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ResumeSubscriptionOutputBody]
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
) -> ErrorModel | ResumeSubscriptionOutputBody | None:
    """Resume a paused subscription

     Lets new deliveries be attempted again. Deliveries recorded while paused stay paused and are never
    sent automatically: the response says how many there are and when the pause began, so you can retry
    all of them or only those since a time with the delivery retry endpoints. Idempotent: resuming a
    subscription that is not paused succeeds with no paused_since.

    Portal tokens: allowed, as POST /portal/api/subscriptions/{subscription_id}:resume.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ResumeSubscriptionOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            subscription_id=subscription_id,
            client=client,
        )
    ).parsed
