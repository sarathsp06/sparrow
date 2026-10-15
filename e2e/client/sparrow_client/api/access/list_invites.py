from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_invites_output_body import ListInvitesOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    include_inactive: bool | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["include_inactive"] = include_inactive

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/invites",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ListInvitesOutputBody | None:
    if response.status_code == 200:
        response_200 = ListInvitesOutputBody.from_dict(response.json())

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
) -> Response[ErrorModel | ListInvitesOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    include_inactive: bool | Unset = UNSET,
) -> Response[ErrorModel | ListInvitesOutputBody]:
    """List invites

     Lists invites, newest first. Only pending invites unless include_inactive is set. Secrets are never
    returned. Not paginated.

    Portal tokens: not allowed.

    Args:
        include_inactive (bool | Unset): Also return redeemed, cancelled, and expired invites.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListInvitesOutputBody]
    """

    kwargs = _get_kwargs(
        include_inactive=include_inactive,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    include_inactive: bool | Unset = UNSET,
) -> ErrorModel | ListInvitesOutputBody | None:
    """List invites

     Lists invites, newest first. Only pending invites unless include_inactive is set. Secrets are never
    returned. Not paginated.

    Portal tokens: not allowed.

    Args:
        include_inactive (bool | Unset): Also return redeemed, cancelled, and expired invites.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListInvitesOutputBody
    """

    return sync_detailed(
        client=client,
        include_inactive=include_inactive,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    include_inactive: bool | Unset = UNSET,
) -> Response[ErrorModel | ListInvitesOutputBody]:
    """List invites

     Lists invites, newest first. Only pending invites unless include_inactive is set. Secrets are never
    returned. Not paginated.

    Portal tokens: not allowed.

    Args:
        include_inactive (bool | Unset): Also return redeemed, cancelled, and expired invites.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListInvitesOutputBody]
    """

    kwargs = _get_kwargs(
        include_inactive=include_inactive,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    include_inactive: bool | Unset = UNSET,
) -> ErrorModel | ListInvitesOutputBody | None:
    """List invites

     Lists invites, newest first. Only pending invites unless include_inactive is set. Secrets are never
    returned. Not paginated.

    Portal tokens: not allowed.

    Args:
        include_inactive (bool | Unset): Also return redeemed, cancelled, and expired invites.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListInvitesOutputBody
    """

    return (
        await asyncio_detailed(
            client=client,
            include_inactive=include_inactive,
        )
    ).parsed
