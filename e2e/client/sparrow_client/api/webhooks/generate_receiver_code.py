from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.generate_receiver_code_input_body import GenerateReceiverCodeInputBody
from ...models.generate_receiver_code_output_body import GenerateReceiverCodeOutputBody
from ...types import Response


def _get_kwargs(
    consumer: str,
    webhook_id: str,
    *,
    body: GenerateReceiverCodeInputBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/consumers/{consumer}/webhooks/{webhook_id}:generateReceiverCode".format(
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
) -> ErrorModel | GenerateReceiverCodeOutputBody | None:
    if response.status_code == 200:
        response_200 = GenerateReceiverCodeOutputBody.from_dict(response.json())

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
) -> Response[ErrorModel | GenerateReceiverCodeOutputBody]:
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
    body: GenerateReceiverCodeInputBody,
) -> Response[ErrorModel | GenerateReceiverCodeOutputBody]:
    """Generate receiver code for a webhook

     Generates a ready-to-run receiver project for the webhook in the chosen language and framework (GET
    /v1/capabilities lists them under receiver_code.targets). The project verifies Sparrow's signatures
    on the raw body before parsing (with a copy of Sparrow's tested verification helper), rejects stale
    timestamps, drops redeliveries by webhook-id, routes on X-Sparrow-Subscription-ID then X-Sparrow-
    Event-Type, decodes each subscription's body (its transform rendered against the event type's sample
    decides JSON, XML or text) and ships with tests that replay signed vectors. Secrets are never
    included: the receiver reads SPARROW_WEBHOOK_SECRET, and the tests use throwaway credentials.
    Nothing is stored. The same webhook, subscriptions and Sparrow version always produce the same
    files; the generated code itself may improve between releases.

    400 for an unsupported target or a subscription that is not on the webhook, 404 for an unknown
    webhook.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:generateReceiverCode.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (GenerateReceiverCodeInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | GenerateReceiverCodeOutputBody]
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
    body: GenerateReceiverCodeInputBody,
) -> ErrorModel | GenerateReceiverCodeOutputBody | None:
    """Generate receiver code for a webhook

     Generates a ready-to-run receiver project for the webhook in the chosen language and framework (GET
    /v1/capabilities lists them under receiver_code.targets). The project verifies Sparrow's signatures
    on the raw body before parsing (with a copy of Sparrow's tested verification helper), rejects stale
    timestamps, drops redeliveries by webhook-id, routes on X-Sparrow-Subscription-ID then X-Sparrow-
    Event-Type, decodes each subscription's body (its transform rendered against the event type's sample
    decides JSON, XML or text) and ships with tests that replay signed vectors. Secrets are never
    included: the receiver reads SPARROW_WEBHOOK_SECRET, and the tests use throwaway credentials.
    Nothing is stored. The same webhook, subscriptions and Sparrow version always produce the same
    files; the generated code itself may improve between releases.

    400 for an unsupported target or a subscription that is not on the webhook, 404 for an unknown
    webhook.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:generateReceiverCode.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (GenerateReceiverCodeInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | GenerateReceiverCodeOutputBody
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
    body: GenerateReceiverCodeInputBody,
) -> Response[ErrorModel | GenerateReceiverCodeOutputBody]:
    """Generate receiver code for a webhook

     Generates a ready-to-run receiver project for the webhook in the chosen language and framework (GET
    /v1/capabilities lists them under receiver_code.targets). The project verifies Sparrow's signatures
    on the raw body before parsing (with a copy of Sparrow's tested verification helper), rejects stale
    timestamps, drops redeliveries by webhook-id, routes on X-Sparrow-Subscription-ID then X-Sparrow-
    Event-Type, decodes each subscription's body (its transform rendered against the event type's sample
    decides JSON, XML or text) and ships with tests that replay signed vectors. Secrets are never
    included: the receiver reads SPARROW_WEBHOOK_SECRET, and the tests use throwaway credentials.
    Nothing is stored. The same webhook, subscriptions and Sparrow version always produce the same
    files; the generated code itself may improve between releases.

    400 for an unsupported target or a subscription that is not on the webhook, 404 for an unknown
    webhook.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:generateReceiverCode.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (GenerateReceiverCodeInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | GenerateReceiverCodeOutputBody]
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
    body: GenerateReceiverCodeInputBody,
) -> ErrorModel | GenerateReceiverCodeOutputBody | None:
    """Generate receiver code for a webhook

     Generates a ready-to-run receiver project for the webhook in the chosen language and framework (GET
    /v1/capabilities lists them under receiver_code.targets). The project verifies Sparrow's signatures
    on the raw body before parsing (with a copy of Sparrow's tested verification helper), rejects stale
    timestamps, drops redeliveries by webhook-id, routes on X-Sparrow-Subscription-ID then X-Sparrow-
    Event-Type, decodes each subscription's body (its transform rendered against the event type's sample
    decides JSON, XML or text) and ships with tests that replay signed vectors. Secrets are never
    included: the receiver reads SPARROW_WEBHOOK_SECRET, and the tests use throwaway credentials.
    Nothing is stored. The same webhook, subscriptions and Sparrow version always produce the same
    files; the generated code itself may improve between releases.

    400 for an unsupported target or a subscription that is not on the webhook, 404 for an unknown
    webhook.

    Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:generateReceiverCode.

    Args:
        consumer (str): Tenant consumer the webhook belongs to.
        webhook_id (str): Webhook id (UUID).
        body (GenerateReceiverCodeInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | GenerateReceiverCodeOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            webhook_id=webhook_id,
            client=client,
            body=body,
        )
    ).parsed
