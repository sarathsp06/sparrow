from http import HTTPStatus
from typing import Any, cast

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs() -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/tls/crls:refresh",
    }

    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | ErrorModel | None:
    if response.status_code == 202:
        response_202 = cast(Any, None)
        return response_202

    if response.status_code == 409:
        response_409 = ErrorModel.from_dict(response.json())

        return response_409

    if response.status_code == 429:
        response_429 = ErrorModel.from_dict(response.json())

        return response_429

    if response.status_code == 500:
        response_500 = ErrorModel.from_dict(response.json())

        return response_500

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
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | ErrorModel]:
    """Refresh cached CRLs on every instance

     Asks every instance (through Postgres NOTIFY) to download its cached CRLs now, for when a CA revokes
    an intermediate and deliveries should stop trusting it before SPARROW_CRL_MAX_AGE passes. Returns
    202 once the request is sent; check listCrls for the result. One request a minute on each server:
    more answer 429. 409 when SPARROW_TLS_REVOCATION=off.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ErrorModel]
    """

    kwargs = _get_kwargs()

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
) -> Any | ErrorModel | None:
    """Refresh cached CRLs on every instance

     Asks every instance (through Postgres NOTIFY) to download its cached CRLs now, for when a CA revokes
    an intermediate and deliveries should stop trusting it before SPARROW_CRL_MAX_AGE passes. Returns
    202 once the request is sent; check listCrls for the result. One request a minute on each server:
    more answer 429. 409 when SPARROW_TLS_REVOCATION=off.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ErrorModel
    """

    return sync_detailed(
        client=client,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | ErrorModel]:
    """Refresh cached CRLs on every instance

     Asks every instance (through Postgres NOTIFY) to download its cached CRLs now, for when a CA revokes
    an intermediate and deliveries should stop trusting it before SPARROW_CRL_MAX_AGE passes. Returns
    202 once the request is sent; check listCrls for the result. One request a minute on each server:
    more answer 429. 409 when SPARROW_TLS_REVOCATION=off.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ErrorModel]
    """

    kwargs = _get_kwargs()

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
) -> Any | ErrorModel | None:
    """Refresh cached CRLs on every instance

     Asks every instance (through Postgres NOTIFY) to download its cached CRLs now, for when a CA revokes
    an intermediate and deliveries should stop trusting it before SPARROW_CRL_MAX_AGE passes. Returns
    202 once the request is sent; check listCrls for the result. One request a minute on each server:
    more answer 429. 409 when SPARROW_TLS_REVOCATION=off.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ErrorModel
    """

    return (
        await asyncio_detailed(
            client=client,
        )
    ).parsed
