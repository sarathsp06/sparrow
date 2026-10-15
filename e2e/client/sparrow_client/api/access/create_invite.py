from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.create_invite_body import CreateInviteBody
from ...models.create_invite_output_body import CreateInviteOutputBody
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs(
    *,
    body: CreateInviteBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/invites",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> CreateInviteOutputBody | ErrorModel | None:
    if response.status_code == 201:
        response_201 = CreateInviteOutputBody.from_dict(response.json())

        return response_201

    if response.status_code == 400:
        response_400 = ErrorModel.from_dict(response.json())

        return response_400

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
) -> Response[CreateInviteOutputBody | ErrorModel]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreateInviteBody,
) -> Response[CreateInviteOutputBody | ErrorModel]:
    """Invite someone with a one-time link

     Creates a single-use, expiring invite. Opening the returned path in the UI exchanges it for a new
    access token named after the invitee (POST /invite/redeem), so nobody pastes a key into chat. A
    consumer invite opens that consumer's portal instead of the operator console.

    Args:
        body (CreateInviteBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CreateInviteOutputBody | ErrorModel]
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
    body: CreateInviteBody,
) -> CreateInviteOutputBody | ErrorModel | None:
    """Invite someone with a one-time link

     Creates a single-use, expiring invite. Opening the returned path in the UI exchanges it for a new
    access token named after the invitee (POST /invite/redeem), so nobody pastes a key into chat. A
    consumer invite opens that consumer's portal instead of the operator console.

    Args:
        body (CreateInviteBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CreateInviteOutputBody | ErrorModel
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreateInviteBody,
) -> Response[CreateInviteOutputBody | ErrorModel]:
    """Invite someone with a one-time link

     Creates a single-use, expiring invite. Opening the returned path in the UI exchanges it for a new
    access token named after the invitee (POST /invite/redeem), so nobody pastes a key into chat. A
    consumer invite opens that consumer's portal instead of the operator console.

    Args:
        body (CreateInviteBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CreateInviteOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: CreateInviteBody,
) -> CreateInviteOutputBody | ErrorModel | None:
    """Invite someone with a one-time link

     Creates a single-use, expiring invite. Opening the returned path in the UI exchanges it for a new
    access token named after the invitee (POST /invite/redeem), so nobody pastes a key into chat. A
    consumer invite opens that consumer's portal instead of the operator console.

    Args:
        body (CreateInviteBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CreateInviteOutputBody | ErrorModel
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
