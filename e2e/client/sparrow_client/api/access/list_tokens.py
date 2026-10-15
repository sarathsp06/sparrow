from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_tokens_output_body import ListTokensOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    consumer: str | Unset = UNSET,
    include_inactive: bool | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["consumer"] = consumer

    params["include_inactive"] = include_inactive

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/tokens",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ListTokensOutputBody | None:
    if response.status_code == 200:
        response_200 = ListTokensOutputBody.from_dict(response.json())

        return response_200

    if response.status_code == 422:
        response_422 = ErrorModel.from_dict(response.json())

        return response_422

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
) -> Response[ErrorModel | ListTokensOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    include_inactive: bool | Unset = UNSET,
) -> Response[ErrorModel | ListTokensOutputBody]:
    """List access tokens

     Lists access tokens, newest first. Secrets are never returned. Revoked and expired tokens are hidden
    unless include_inactive is set. Not paginated.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Only tokens limited to this consumer. Omit to list every token,
            tenant-wide ones included.
        include_inactive (bool | Unset): Also return revoked and expired tokens.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListTokensOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        include_inactive=include_inactive,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    include_inactive: bool | Unset = UNSET,
) -> ErrorModel | ListTokensOutputBody | None:
    """List access tokens

     Lists access tokens, newest first. Secrets are never returned. Revoked and expired tokens are hidden
    unless include_inactive is set. Not paginated.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Only tokens limited to this consumer. Omit to list every token,
            tenant-wide ones included.
        include_inactive (bool | Unset): Also return revoked and expired tokens.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListTokensOutputBody
    """

    return sync_detailed(
        client=client,
        consumer=consumer,
        include_inactive=include_inactive,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    include_inactive: bool | Unset = UNSET,
) -> Response[ErrorModel | ListTokensOutputBody]:
    """List access tokens

     Lists access tokens, newest first. Secrets are never returned. Revoked and expired tokens are hidden
    unless include_inactive is set. Not paginated.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Only tokens limited to this consumer. Omit to list every token,
            tenant-wide ones included.
        include_inactive (bool | Unset): Also return revoked and expired tokens.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListTokensOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        include_inactive=include_inactive,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    include_inactive: bool | Unset = UNSET,
) -> ErrorModel | ListTokensOutputBody | None:
    """List access tokens

     Lists access tokens, newest first. Secrets are never returned. Revoked and expired tokens are hidden
    unless include_inactive is set. Not paginated.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Only tokens limited to this consumer. Omit to list every token,
            tenant-wide ones included.
        include_inactive (bool | Unset): Also return revoked and expired tokens.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListTokensOutputBody
    """

    return (
        await asyncio_detailed(
            client=client,
            consumer=consumer,
            include_inactive=include_inactive,
        )
    ).parsed
