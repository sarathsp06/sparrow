from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.event_type_version_item import EventTypeVersionItem
from ...types import Response


def _get_kwargs(
    name: str,
    version: int,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/event-types/{name}/versions/{version}".format(
            name=quote(str(name), safe=""),
            version=quote(str(version), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | EventTypeVersionItem | None:
    if response.status_code == 200:
        response_200 = EventTypeVersionItem.from_dict(response.json())

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
) -> Response[ErrorModel | EventTypeVersionItem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    name: str,
    version: int,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorModel | EventTypeVersionItem]:
    """Get one version of an event type

     Fetches a specific version's schema, sample payload and description.

    Args:
        name (str):
        version (int): Version number.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | EventTypeVersionItem]
    """

    kwargs = _get_kwargs(
        name=name,
        version=version,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    name: str,
    version: int,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorModel | EventTypeVersionItem | None:
    """Get one version of an event type

     Fetches a specific version's schema, sample payload and description.

    Args:
        name (str):
        version (int): Version number.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | EventTypeVersionItem
    """

    return sync_detailed(
        name=name,
        version=version,
        client=client,
    ).parsed


async def asyncio_detailed(
    name: str,
    version: int,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ErrorModel | EventTypeVersionItem]:
    """Get one version of an event type

     Fetches a specific version's schema, sample payload and description.

    Args:
        name (str):
        version (int): Version number.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | EventTypeVersionItem]
    """

    kwargs = _get_kwargs(
        name=name,
        version=version,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    name: str,
    version: int,
    *,
    client: AuthenticatedClient | Client,
) -> ErrorModel | EventTypeVersionItem | None:
    """Get one version of an event type

     Fetches a specific version's schema, sample payload and description.

    Args:
        name (str):
        version (int): Version number.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | EventTypeVersionItem
    """

    return (
        await asyncio_detailed(
            name=name,
            version=version,
            client=client,
        )
    ).parsed
