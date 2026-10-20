from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.send_test_event_input_body import SendTestEventInputBody
from ...models.send_test_event_output_body import SendTestEventOutputBody
from ...types import Response


def _get_kwargs(
    consumer: str,
    webhook_id: str,
    *,
    body: SendTestEventInputBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/consumers/{consumer}/webhooks/{webhook_id}:sendTestEvent".format(
            consumer=quote(str(consumer), safe=""),
            webhook_id=quote(str(webhook_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | SendTestEventOutputBody | None:
    if response.status_code == 200:
        response_200 = SendTestEventOutputBody.from_dict(response.json())

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

    if response.status_code == 429:
        response_429 = ErrorModel.from_dict(response.json())

        return response_429

    if response.status_code == 500:
        response_500 = ErrorModel.from_dict(response.json())

        return response_500

    if response.status_code == 503:
        response_503 = ErrorModel.from_dict(response.json())

        return response_503

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorModel | SendTestEventOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    consumer: str,
    webhook_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: SendTestEventInputBody,
) -> Response[ErrorModel | SendTestEventOutputBody]:
    """Send a test event to a webhook

     Sends one signed delivery of payload, or of the subscription's event type sample when payload is
    omitted (rendered through the subscription's transform, as a real delivery would be), to the
    webhook's URL, synchronously, and returns what the receiver answered. It uses the same request
    builder, signing, network policy (SSRF protection) and timeout as real deliveries and adds the
    header X-Sparrow-Test: true. It creates no delivery and never counts toward the webhook's health or
    auto-disable. A receiver error is reported in the response (status_code, error_class), not as an API
    error.

    One test event per webhook every 2 seconds on each server: more answer 429. 404 for an unknown
    webhook or a subscription not on it.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:sendTestEvent.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (SendTestEventInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | SendTestEventOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    webhook_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: SendTestEventInputBody,
) -> ErrorModel | SendTestEventOutputBody | None:
    """Send a test event to a webhook

     Sends one signed delivery of payload, or of the subscription's event type sample when payload is
    omitted (rendered through the subscription's transform, as a real delivery would be), to the
    webhook's URL, synchronously, and returns what the receiver answered. It uses the same request
    builder, signing, network policy (SSRF protection) and timeout as real deliveries and adds the
    header X-Sparrow-Test: true. It creates no delivery and never counts toward the webhook's health or
    auto-disable. A receiver error is reported in the response (status_code, error_class), not as an API
    error.

    One test event per webhook every 2 seconds on each server: more answer 429. 404 for an unknown
    webhook or a subscription not on it.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:sendTestEvent.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (SendTestEventInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | SendTestEventOutputBody
    """

    return sync_detailed(
        consumer=consumer,
        webhook_id=webhook_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    webhook_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: SendTestEventInputBody,
) -> Response[ErrorModel | SendTestEventOutputBody]:
    """Send a test event to a webhook

     Sends one signed delivery of payload, or of the subscription's event type sample when payload is
    omitted (rendered through the subscription's transform, as a real delivery would be), to the
    webhook's URL, synchronously, and returns what the receiver answered. It uses the same request
    builder, signing, network policy (SSRF protection) and timeout as real deliveries and adds the
    header X-Sparrow-Test: true. It creates no delivery and never counts toward the webhook's health or
    auto-disable. A receiver error is reported in the response (status_code, error_class), not as an API
    error.

    One test event per webhook every 2 seconds on each server: more answer 429. 404 for an unknown
    webhook or a subscription not on it.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:sendTestEvent.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (SendTestEventInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | SendTestEventOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    webhook_id: str,
    *,
    client: AuthenticatedClient | Client,
    body: SendTestEventInputBody,
) -> ErrorModel | SendTestEventOutputBody | None:
    """Send a test event to a webhook

     Sends one signed delivery of payload, or of the subscription's event type sample when payload is
    omitted (rendered through the subscription's transform, as a real delivery would be), to the
    webhook's URL, synchronously, and returns what the receiver answered. It uses the same request
    builder, signing, network policy (SSRF protection) and timeout as real deliveries and adds the
    header X-Sparrow-Test: true. It creates no delivery and never counts toward the webhook's health or
    auto-disable. A receiver error is reported in the response (status_code, error_class), not as an API
    error.

    One test event per webhook every 2 seconds on each server: more answer 429. 404 for an unknown
    webhook or a subscription not on it.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:sendTestEvent.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (SendTestEventInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | SendTestEventOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            webhook_id=webhook_id,
            client=client,
            body=body,
        )
    ).parsed
