from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.push_event_body import PushEventBody
from ...models.push_event_output_body import PushEventOutputBody
from ...types import UNSET, Response


def _get_kwargs(
    consumer: str,
    *,
    body: PushEventBody,
    event: str,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    params: dict[str, Any] = {}

    params["event"] = event

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/consumers/{consumer}/events".format(
            consumer=quote(str(consumer), safe=""),
        ),
        "params": params,
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | PushEventOutputBody | None:
    if response.status_code == 201:
        response_201 = PushEventOutputBody.from_dict(response.json())

        return response_201

    if response.status_code == 400:
        response_400 = ErrorModel.from_dict(response.json())

        return response_400

    if response.status_code == 404:
        response_404 = ErrorModel.from_dict(response.json())

        return response_404

    if response.status_code == 409:
        response_409 = ErrorModel.from_dict(response.json())

        return response_409

    if response.status_code == 413:
        response_413 = ErrorModel.from_dict(response.json())

        return response_413

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
) -> Response[ErrorModel | PushEventOutputBody]:
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
    body: PushEventBody,
    event: str,
) -> Response[ErrorModel | PushEventOutputBody]:
    """Push an event occurrence for asynchronous delivery

     Records one occurrence of a registered event type and asynchronously fans it out to every
    subscription whose event name and label filters match. Returns 201 as soon as the event is stored
    and queued; delivery happens in the background, so check Deliveries for outcomes. A payload that
    fails the event type's JSON Schema is still accepted (schema_valid: false, with warnings).

    Idempotency: with idempotency_key, a repeat push with the same key in the same consumer returns 201
    with duplicate: true and the original event_id, and creates nothing. Without a key every call
    creates a new event.

    Failures: 404 for an unregistered event type (unless SPARROW_AUTO_REGISTER_EVENTS=true), 409 for an
    inactive one, 400 for a reserved sparrow.* name, a name over 255 characters or invalid labels, 413
    for a body over SPARROW_MAX_BODY_BYTES.

    Portal tokens: not allowed (pushing is the producer's job).

    Args:
        consumer (str): Consumer to record the occurrence in. Created implicitly on first use.
        event (str): Name of a registered, active event type (at most 255 characters). An
            unregistered name is 404 unless SPARROW_AUTO_REGISTER_EVENTS=true, which registers it if
            it follows the event type naming rule (lowercase letters, digits, '_' and '-' in dot-
            separated parts; 400 otherwise); sparrow.* names are reserved (400).
        body (PushEventBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | PushEventOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        body=body,
        event=event,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    body: PushEventBody,
    event: str,
) -> ErrorModel | PushEventOutputBody | None:
    """Push an event occurrence for asynchronous delivery

     Records one occurrence of a registered event type and asynchronously fans it out to every
    subscription whose event name and label filters match. Returns 201 as soon as the event is stored
    and queued; delivery happens in the background, so check Deliveries for outcomes. A payload that
    fails the event type's JSON Schema is still accepted (schema_valid: false, with warnings).

    Idempotency: with idempotency_key, a repeat push with the same key in the same consumer returns 201
    with duplicate: true and the original event_id, and creates nothing. Without a key every call
    creates a new event.

    Failures: 404 for an unregistered event type (unless SPARROW_AUTO_REGISTER_EVENTS=true), 409 for an
    inactive one, 400 for a reserved sparrow.* name, a name over 255 characters or invalid labels, 413
    for a body over SPARROW_MAX_BODY_BYTES.

    Portal tokens: not allowed (pushing is the producer's job).

    Args:
        consumer (str): Consumer to record the occurrence in. Created implicitly on first use.
        event (str): Name of a registered, active event type (at most 255 characters). An
            unregistered name is 404 unless SPARROW_AUTO_REGISTER_EVENTS=true, which registers it if
            it follows the event type naming rule (lowercase letters, digits, '_' and '-' in dot-
            separated parts; 400 otherwise); sparrow.* names are reserved (400).
        body (PushEventBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | PushEventOutputBody
    """

    return sync_detailed(
        consumer=consumer,
        client=client,
        body=body,
        event=event,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    body: PushEventBody,
    event: str,
) -> Response[ErrorModel | PushEventOutputBody]:
    """Push an event occurrence for asynchronous delivery

     Records one occurrence of a registered event type and asynchronously fans it out to every
    subscription whose event name and label filters match. Returns 201 as soon as the event is stored
    and queued; delivery happens in the background, so check Deliveries for outcomes. A payload that
    fails the event type's JSON Schema is still accepted (schema_valid: false, with warnings).

    Idempotency: with idempotency_key, a repeat push with the same key in the same consumer returns 201
    with duplicate: true and the original event_id, and creates nothing. Without a key every call
    creates a new event.

    Failures: 404 for an unregistered event type (unless SPARROW_AUTO_REGISTER_EVENTS=true), 409 for an
    inactive one, 400 for a reserved sparrow.* name, a name over 255 characters or invalid labels, 413
    for a body over SPARROW_MAX_BODY_BYTES.

    Portal tokens: not allowed (pushing is the producer's job).

    Args:
        consumer (str): Consumer to record the occurrence in. Created implicitly on first use.
        event (str): Name of a registered, active event type (at most 255 characters). An
            unregistered name is 404 unless SPARROW_AUTO_REGISTER_EVENTS=true, which registers it if
            it follows the event type naming rule (lowercase letters, digits, '_' and '-' in dot-
            separated parts; 400 otherwise); sparrow.* names are reserved (400).
        body (PushEventBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | PushEventOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        body=body,
        event=event,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    body: PushEventBody,
    event: str,
) -> ErrorModel | PushEventOutputBody | None:
    """Push an event occurrence for asynchronous delivery

     Records one occurrence of a registered event type and asynchronously fans it out to every
    subscription whose event name and label filters match. Returns 201 as soon as the event is stored
    and queued; delivery happens in the background, so check Deliveries for outcomes. A payload that
    fails the event type's JSON Schema is still accepted (schema_valid: false, with warnings).

    Idempotency: with idempotency_key, a repeat push with the same key in the same consumer returns 201
    with duplicate: true and the original event_id, and creates nothing. Without a key every call
    creates a new event.

    Failures: 404 for an unregistered event type (unless SPARROW_AUTO_REGISTER_EVENTS=true), 409 for an
    inactive one, 400 for a reserved sparrow.* name, a name over 255 characters or invalid labels, 413
    for a body over SPARROW_MAX_BODY_BYTES.

    Portal tokens: not allowed (pushing is the producer's job).

    Args:
        consumer (str): Consumer to record the occurrence in. Created implicitly on first use.
        event (str): Name of a registered, active event type (at most 255 characters). An
            unregistered name is 404 unless SPARROW_AUTO_REGISTER_EVENTS=true, which registers it if
            it follows the event type naming rule (lowercase letters, digits, '_' and '-' in dot-
            separated parts; 400 otherwise); sparrow.* names are reserved (400).
        body (PushEventBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | PushEventOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            client=client,
            body=body,
            event=event,
        )
    ).parsed
