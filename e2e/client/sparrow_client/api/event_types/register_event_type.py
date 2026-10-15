from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.event_type_body import EventTypeBody
from ...models.event_type_item import EventTypeItem
from ...types import Response


def _get_kwargs(
    *,
    body: EventTypeBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/event-types",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | EventTypeItem | None:
    if response.status_code == 201:
        response_201 = EventTypeItem.from_dict(response.json())

        return response_201

    if response.status_code == 400:
        response_400 = ErrorModel.from_dict(response.json())

        return response_400

    if response.status_code == 409:
        response_409 = ErrorModel.from_dict(response.json())

        return response_409

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
) -> Response[ErrorModel | EventTypeItem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: EventTypeBody,
) -> Response[ErrorModel | EventTypeItem]:
    """Register an event type definition

     Defines a named event type at version 1, optionally with a JSON Schema; the response includes a
    sample_payload generated from the schema. Pushing an event requires its type to be registered
    (unless SPARROW_AUTO_REGISTER_EVENTS=true). Event types are never deleted; deactivate one with
    active: false.

    Not idempotent: 409 when the name already exists (use PATCH, or :import, to change it). 400 for a
    missing, too long or reserved (sparrow.*) name or an invalid schema.

    Portal tokens: not allowed.

    Args:
        body (EventTypeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | EventTypeItem]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    body: EventTypeBody,
) -> ErrorModel | EventTypeItem | None:
    """Register an event type definition

     Defines a named event type at version 1, optionally with a JSON Schema; the response includes a
    sample_payload generated from the schema. Pushing an event requires its type to be registered
    (unless SPARROW_AUTO_REGISTER_EVENTS=true). Event types are never deleted; deactivate one with
    active: false.

    Not idempotent: 409 when the name already exists (use PATCH, or :import, to change it). 400 for a
    missing, too long or reserved (sparrow.*) name or an invalid schema.

    Portal tokens: not allowed.

    Args:
        body (EventTypeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | EventTypeItem
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: EventTypeBody,
) -> Response[ErrorModel | EventTypeItem]:
    """Register an event type definition

     Defines a named event type at version 1, optionally with a JSON Schema; the response includes a
    sample_payload generated from the schema. Pushing an event requires its type to be registered
    (unless SPARROW_AUTO_REGISTER_EVENTS=true). Event types are never deleted; deactivate one with
    active: false.

    Not idempotent: 409 when the name already exists (use PATCH, or :import, to change it). 400 for a
    missing, too long or reserved (sparrow.*) name or an invalid schema.

    Portal tokens: not allowed.

    Args:
        body (EventTypeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | EventTypeItem]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: EventTypeBody,
) -> ErrorModel | EventTypeItem | None:
    """Register an event type definition

     Defines a named event type at version 1, optionally with a JSON Schema; the response includes a
    sample_payload generated from the schema. Pushing an event requires its type to be registered
    (unless SPARROW_AUTO_REGISTER_EVENTS=true). Event types are never deleted; deactivate one with
    active: false.

    Not idempotent: 409 when the name already exists (use PATCH, or :import, to change it). 400 for a
    missing, too long or reserved (sparrow.*) name or an invalid schema.

    Portal tokens: not allowed.

    Args:
        body (EventTypeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | EventTypeItem
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
