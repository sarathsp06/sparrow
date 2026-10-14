from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_consumers_output_body import ListConsumersOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    q: str | Unset = UNSET,
    limit: int | Unset = 50,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["q"] = q

    params["limit"] = limit

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/consumers",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ListConsumersOutputBody:
    if response.status_code == 200:
        response_200 = ListConsumersOutputBody.from_dict(response.json())

        return response_200

    response_default = ErrorModel.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorModel | ListConsumersOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    limit: int | Unset = 50,
) -> Response[ErrorModel | ListConsumersOutputBody]:
    """Search consumer names

     Lists the consumers that own a webhook or an event, sorted by name, optionally only those whose name
    contains q (case-insensitive). Consumers are created implicitly by registering a webhook or pushing
    an event under them; this is how clients find them, e.g. for a consumer picker.

    Args:
        q (str | Unset): Only consumers whose name contains this text, case-insensitive.
        limit (int | Unset): Maximum names to return. Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListConsumersOutputBody]
    """

    kwargs = _get_kwargs(
        q=q,
        limit=limit,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    limit: int | Unset = 50,
) -> ErrorModel | ListConsumersOutputBody | None:
    """Search consumer names

     Lists the consumers that own a webhook or an event, sorted by name, optionally only those whose name
    contains q (case-insensitive). Consumers are created implicitly by registering a webhook or pushing
    an event under them; this is how clients find them, e.g. for a consumer picker.

    Args:
        q (str | Unset): Only consumers whose name contains this text, case-insensitive.
        limit (int | Unset): Maximum names to return. Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListConsumersOutputBody
    """

    return sync_detailed(
        client=client,
        q=q,
        limit=limit,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    limit: int | Unset = 50,
) -> Response[ErrorModel | ListConsumersOutputBody]:
    """Search consumer names

     Lists the consumers that own a webhook or an event, sorted by name, optionally only those whose name
    contains q (case-insensitive). Consumers are created implicitly by registering a webhook or pushing
    an event under them; this is how clients find them, e.g. for a consumer picker.

    Args:
        q (str | Unset): Only consumers whose name contains this text, case-insensitive.
        limit (int | Unset): Maximum names to return. Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListConsumersOutputBody]
    """

    kwargs = _get_kwargs(
        q=q,
        limit=limit,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    limit: int | Unset = 50,
) -> ErrorModel | ListConsumersOutputBody | None:
    """Search consumer names

     Lists the consumers that own a webhook or an event, sorted by name, optionally only those whose name
    contains q (case-insensitive). Consumers are created implicitly by registering a webhook or pushing
    an event under them; this is how clients find them, e.g. for a consumer picker.

    Args:
        q (str | Unset): Only consumers whose name contains this text, case-insensitive.
        limit (int | Unset): Maximum names to return. Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListConsumersOutputBody
    """

    return (
        await asyncio_detailed(
            client=client,
            q=q,
            limit=limit,
        )
    ).parsed
