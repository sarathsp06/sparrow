from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.patch_subscription_body import PatchSubscriptionBody
from ...models.subscription_item import SubscriptionItem
from ...types import Response


def _get_kwargs(
    consumer: str,
    subscription_id: str,
    *,
    body: PatchSubscriptionBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/v1/consumers/{consumer}/subscriptions/{subscription_id}".format(
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
    body: PatchSubscriptionBody,
) -> Response[ErrorModel | SubscriptionItem]:
    """Partially update a subscription

     Partially updates a subscription's headers, method, timeout, transform, or label filters: only
    fields present in the body change, and map fields replace the whole map. The linked webhook_id and
    event_name cannot be changed; delete and recreate instead. A changed transform_template is recorded
    as a new template version with template_source and template_notes. Returns the updated subscription.
    Idempotent: repeating the same body records no new version.

    400 for invalid values, or when the webhook has requires_transform and the result has no enabled
    template. 404 when it is not in this consumer.

    Portal tokens: allowed, as PATCH /portal/api/subscriptions/{subscription_id}.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).
        body (PatchSubscriptionBody):

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
    body: PatchSubscriptionBody,
) -> ErrorModel | SubscriptionItem | None:
    """Partially update a subscription

     Partially updates a subscription's headers, method, timeout, transform, or label filters: only
    fields present in the body change, and map fields replace the whole map. The linked webhook_id and
    event_name cannot be changed; delete and recreate instead. A changed transform_template is recorded
    as a new template version with template_source and template_notes. Returns the updated subscription.
    Idempotent: repeating the same body records no new version.

    400 for invalid values, or when the webhook has requires_transform and the result has no enabled
    template. 404 when it is not in this consumer.

    Portal tokens: allowed, as PATCH /portal/api/subscriptions/{subscription_id}.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).
        body (PatchSubscriptionBody):

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
    body: PatchSubscriptionBody,
) -> Response[ErrorModel | SubscriptionItem]:
    """Partially update a subscription

     Partially updates a subscription's headers, method, timeout, transform, or label filters: only
    fields present in the body change, and map fields replace the whole map. The linked webhook_id and
    event_name cannot be changed; delete and recreate instead. A changed transform_template is recorded
    as a new template version with template_source and template_notes. Returns the updated subscription.
    Idempotent: repeating the same body records no new version.

    400 for invalid values, or when the webhook has requires_transform and the result has no enabled
    template. 404 when it is not in this consumer.

    Portal tokens: allowed, as PATCH /portal/api/subscriptions/{subscription_id}.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).
        body (PatchSubscriptionBody):

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
    body: PatchSubscriptionBody,
) -> ErrorModel | SubscriptionItem | None:
    """Partially update a subscription

     Partially updates a subscription's headers, method, timeout, transform, or label filters: only
    fields present in the body change, and map fields replace the whole map. The linked webhook_id and
    event_name cannot be changed; delete and recreate instead. A changed transform_template is recorded
    as a new template version with template_source and template_notes. Returns the updated subscription.
    Idempotent: repeating the same body records no new version.

    400 for invalid values, or when the webhook has requires_transform and the result has no enabled
    template. 404 when it is not in this consumer.

    Portal tokens: allowed, as PATCH /portal/api/subscriptions/{subscription_id}.

    Args:
        consumer (str): Consumer the subscription belongs to.
        subscription_id (str): Subscription id (UUID).
        body (PatchSubscriptionBody):

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
