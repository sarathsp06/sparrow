from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs(
    token_id: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/v1/tokens/{token_id}".format(
            token_id=quote(str(token_id), safe=""),
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
    token_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | ErrorModel]:
    """Revoke an access token

     Revokes a token: it stops working immediately on this server and within 30 seconds on other
    instances. Revoking is permanent and idempotent (204 again for an already revoked token). Returns
    204; 404 for an unknown id.

    Portal tokens: not allowed.

    Args:
        token_id (str): Token id (the id field, not the secret).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ErrorModel]
    """

    kwargs = _get_kwargs(
        token_id=token_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    token_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Any | ErrorModel | None:
    """Revoke an access token

     Revokes a token: it stops working immediately on this server and within 30 seconds on other
    instances. Revoking is permanent and idempotent (204 again for an already revoked token). Returns
    204; 404 for an unknown id.

    Portal tokens: not allowed.

    Args:
        token_id (str): Token id (the id field, not the secret).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ErrorModel
    """

    return sync_detailed(
        token_id=token_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    token_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | ErrorModel]:
    """Revoke an access token

     Revokes a token: it stops working immediately on this server and within 30 seconds on other
    instances. Revoking is permanent and idempotent (204 again for an already revoked token). Returns
    204; 404 for an unknown id.

    Portal tokens: not allowed.

    Args:
        token_id (str): Token id (the id field, not the secret).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ErrorModel]
    """

    kwargs = _get_kwargs(
        token_id=token_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    token_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Any | ErrorModel | None:
    """Revoke an access token

     Revokes a token: it stops working immediately on this server and within 30 seconds on other
    instances. Revoking is permanent and idempotent (204 again for an already revoked token). Returns
    204; 404 for an unknown id.

    Portal tokens: not allowed.

    Args:
        token_id (str): Token id (the id field, not the secret).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ErrorModel
    """

    return (
        await asyncio_detailed(
            token_id=token_id,
            client=client,
        )
    ).parsed
