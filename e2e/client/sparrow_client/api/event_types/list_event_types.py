from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_event_types_output_body import ListEventTypesOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    active_only: bool | Unset = False,
    consumer: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["active_only"] = active_only

    params["consumer"] = consumer

    params["limit"] = limit

    params["offset"] = offset

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/event-types",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ListEventTypesOutputBody:
    if response.status_code == 200:
        response_200 = ListEventTypesOutputBody.from_dict(response.json())

        return response_200

    response_default = ErrorModel.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorModel | ListEventTypesOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    active_only: bool | Unset = False,
    consumer: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListEventTypesOutputBody]:
    """List event type definitions

     Lists registered event types, optionally only active ones. Sparrow's own sparrow.* events are listed
    only with consumer=_sparrow.

    Args:
        active_only (bool | Unset): Only return active event types. Default: False.
        consumer (str | Unset): Return the event types this consumer can subscribe to: Sparrow's
            own sparrow.* events for the _sparrow consumer, tenant event types for any other consumer
            or when omitted.
        limit (int | Unset): Maximum items to return. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListEventTypesOutputBody]
    """

    kwargs = _get_kwargs(
        active_only=active_only,
        consumer=consumer,
        limit=limit,
        offset=offset,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    active_only: bool | Unset = False,
    consumer: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> ErrorModel | ListEventTypesOutputBody | None:
    """List event type definitions

     Lists registered event types, optionally only active ones. Sparrow's own sparrow.* events are listed
    only with consumer=_sparrow.

    Args:
        active_only (bool | Unset): Only return active event types. Default: False.
        consumer (str | Unset): Return the event types this consumer can subscribe to: Sparrow's
            own sparrow.* events for the _sparrow consumer, tenant event types for any other consumer
            or when omitted.
        limit (int | Unset): Maximum items to return. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListEventTypesOutputBody
    """

    return sync_detailed(
        client=client,
        active_only=active_only,
        consumer=consumer,
        limit=limit,
        offset=offset,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    active_only: bool | Unset = False,
    consumer: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListEventTypesOutputBody]:
    """List event type definitions

     Lists registered event types, optionally only active ones. Sparrow's own sparrow.* events are listed
    only with consumer=_sparrow.

    Args:
        active_only (bool | Unset): Only return active event types. Default: False.
        consumer (str | Unset): Return the event types this consumer can subscribe to: Sparrow's
            own sparrow.* events for the _sparrow consumer, tenant event types for any other consumer
            or when omitted.
        limit (int | Unset): Maximum items to return. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListEventTypesOutputBody]
    """

    kwargs = _get_kwargs(
        active_only=active_only,
        consumer=consumer,
        limit=limit,
        offset=offset,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    active_only: bool | Unset = False,
    consumer: str | Unset = UNSET,
    limit: int | Unset = 50,
    offset: int | Unset = 0,
) -> ErrorModel | ListEventTypesOutputBody | None:
    """List event type definitions

     Lists registered event types, optionally only active ones. Sparrow's own sparrow.* events are listed
    only with consumer=_sparrow.

    Args:
        active_only (bool | Unset): Only return active event types. Default: False.
        consumer (str | Unset): Return the event types this consumer can subscribe to: Sparrow's
            own sparrow.* events for the _sparrow consumer, tenant event types for any other consumer
            or when omitted.
        limit (int | Unset): Maximum items to return. Default: 50.
        offset (int | Unset): Number of items to skip, for pagination. Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListEventTypesOutputBody
    """

    return (
        await asyncio_detailed(
            client=client,
            active_only=active_only,
            consumer=consumer,
            limit=limit,
            offset=offset,
        )
    ).parsed
