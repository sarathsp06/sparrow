from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.event_type_bundle import EventTypeBundle
from ...models.export_event_types_input_body import ExportEventTypesInputBody
from ...types import Response


def _get_kwargs(
    *,
    body: ExportEventTypesInputBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/event-types:export",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | EventTypeBundle | None:
    if response.status_code == 200:
        response_200 = EventTypeBundle.from_dict(response.json())

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
) -> Response[ErrorModel | EventTypeBundle]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: ExportEventTypesInputBody,
) -> Response[ErrorModel | EventTypeBundle]:
    """Export event type definitions as a bundle

     Returns the current definitions of the selected event types as one JSON bundle, ready to import into
    another environment with importEventTypes. Choose exactly one of names, prefix or all. An unknown
    name fails the whole export. Sparrow's own sparrow.* event types are never exported. Read-only
    despite POST (the selection travels in the body).

    Portal tokens: not allowed.

    Args:
        body (ExportEventTypesInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | EventTypeBundle]
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
    body: ExportEventTypesInputBody,
) -> ErrorModel | EventTypeBundle | None:
    """Export event type definitions as a bundle

     Returns the current definitions of the selected event types as one JSON bundle, ready to import into
    another environment with importEventTypes. Choose exactly one of names, prefix or all. An unknown
    name fails the whole export. Sparrow's own sparrow.* event types are never exported. Read-only
    despite POST (the selection travels in the body).

    Portal tokens: not allowed.

    Args:
        body (ExportEventTypesInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | EventTypeBundle
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: ExportEventTypesInputBody,
) -> Response[ErrorModel | EventTypeBundle]:
    """Export event type definitions as a bundle

     Returns the current definitions of the selected event types as one JSON bundle, ready to import into
    another environment with importEventTypes. Choose exactly one of names, prefix or all. An unknown
    name fails the whole export. Sparrow's own sparrow.* event types are never exported. Read-only
    despite POST (the selection travels in the body).

    Portal tokens: not allowed.

    Args:
        body (ExportEventTypesInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | EventTypeBundle]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: ExportEventTypesInputBody,
) -> ErrorModel | EventTypeBundle | None:
    """Export event type definitions as a bundle

     Returns the current definitions of the selected event types as one JSON bundle, ready to import into
    another environment with importEventTypes. Choose exactly one of names, prefix or all. An unknown
    name fails the whole export. Sparrow's own sparrow.* event types are never exported. Read-only
    despite POST (the selection travels in the body).

    Portal tokens: not allowed.

    Args:
        body (ExportEventTypesInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | EventTypeBundle
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
