from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs(
    invite_id: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/v1/invites/{invite_id}".format(
            invite_id=quote(str(invite_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | ErrorModel | None:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

    if response.status_code == 404:
        response_404 = ErrorModel.from_dict(response.json())

        return response_404

    if response.status_code == 409:
        response_409 = ErrorModel.from_dict(response.json())

        return response_409

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


def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | ErrorModel]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    invite_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | ErrorModel]:
    """Cancel a pending invite

     Cancels a pending invite so it can no longer be redeemed. Returns 204. 409 when the invite is no
    longer pending (redeemed, cancelled or expired); a token already created from it keeps working, so
    revoke that separately. 404 for an unknown id.

    Portal tokens: not allowed.

    Args:
        invite_id (str): Invite id (the id field, not the secret).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ErrorModel]
    """

    kwargs = _get_kwargs(
        invite_id=invite_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    invite_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Any | ErrorModel | None:
    """Cancel a pending invite

     Cancels a pending invite so it can no longer be redeemed. Returns 204. 409 when the invite is no
    longer pending (redeemed, cancelled or expired); a token already created from it keeps working, so
    revoke that separately. 404 for an unknown id.

    Portal tokens: not allowed.

    Args:
        invite_id (str): Invite id (the id field, not the secret).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ErrorModel
    """

    return sync_detailed(
        invite_id=invite_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    invite_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | ErrorModel]:
    """Cancel a pending invite

     Cancels a pending invite so it can no longer be redeemed. Returns 204. 409 when the invite is no
    longer pending (redeemed, cancelled or expired); a token already created from it keeps working, so
    revoke that separately. 404 for an unknown id.

    Portal tokens: not allowed.

    Args:
        invite_id (str): Invite id (the id field, not the secret).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ErrorModel]
    """

    kwargs = _get_kwargs(
        invite_id=invite_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    invite_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Any | ErrorModel | None:
    """Cancel a pending invite

     Cancels a pending invite so it can no longer be redeemed. Returns 204. 409 when the invite is no
    longer pending (redeemed, cancelled or expired); a token already created from it keeps working, so
    revoke that separately. 404 for an unknown id.

    Portal tokens: not allowed.

    Args:
        invite_id (str): Invite id (the id field, not the secret).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ErrorModel
    """

    return (
        await asyncio_detailed(
            invite_id=invite_id,
            client=client,
        )
    ).parsed
