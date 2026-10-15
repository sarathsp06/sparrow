from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.event_type_item import EventTypeItem
from ...types import Response


def _get_kwargs(
    name: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/event-types/{name}".format(
            name=quote(str(name), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | EventTypeItem | None:
    if response.status_code == 200:
        response_200 = EventTypeItem.from_dict(response.json())

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
) -> Response[ErrorModel | EventTypeItem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    name: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorModel | EventTypeItem]:
    """Get an event type by name

     Returns one event type's current version: schema, sample payload, and metadata. 404 when the name is
    not registered.

    Portal tokens: allowed, as GET /portal/api/event-types/{name}.

    Args:
        name (str): Event type name, e.g. order.created.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | EventTypeItem]
    """

    kwargs = _get_kwargs(
        name=name,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    name: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorModel | EventTypeItem | None:
    """Get an event type by name

     Returns one event type's current version: schema, sample payload, and metadata. 404 when the name is
    not registered.

    Portal tokens: allowed, as GET /portal/api/event-types/{name}.

    Args:
        name (str): Event type name, e.g. order.created.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | EventTypeItem
    """

    return sync_detailed(
        name=name,
        client=client,
    ).parsed


async def asyncio_detailed(
    name: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorModel | EventTypeItem]:
    """Get an event type by name

     Returns one event type's current version: schema, sample payload, and metadata. 404 when the name is
    not registered.

    Portal tokens: allowed, as GET /portal/api/event-types/{name}.

    Args:
        name (str): Event type name, e.g. order.created.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | EventTypeItem]
    """

    kwargs = _get_kwargs(
        name=name,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    name: str,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorModel | EventTypeItem | None:
    """Get an event type by name

     Returns one event type's current version: schema, sample payload, and metadata. 404 when the name is
    not registered.

    Portal tokens: allowed, as GET /portal/api/event-types/{name}.

    Args:
        name (str): Event type name, e.g. order.created.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | EventTypeItem
    """

    return (
        await asyncio_detailed(
            name=name,
            client=client,
        )
    ).parsed
