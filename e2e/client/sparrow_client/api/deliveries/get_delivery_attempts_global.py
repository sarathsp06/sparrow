from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.attempts_output_body import AttemptsOutputBody
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs(
    delivery_id: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/deliveries/{delivery_id}/attempts".format(
            delivery_id=quote(str(delivery_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> AttemptsOutputBody | ErrorModel | None:
    if response.status_code == 200:
        response_200 = AttemptsOutputBody.from_dict(response.json())

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
) -> Response[AttemptsOutputBody | ErrorModel]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    delivery_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[AttemptsOutputBody | ErrorModel]:
    """Get a delivery's per-attempt history (any consumer)

     Same as getDeliveryAttempts, without a consumer.

    Portal tokens: not allowed.

    Args:
        delivery_id (str): Delivery id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AttemptsOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs(
        delivery_id=delivery_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    delivery_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> AttemptsOutputBody | ErrorModel | None:
    """Get a delivery's per-attempt history (any consumer)

     Same as getDeliveryAttempts, without a consumer.

    Portal tokens: not allowed.

    Args:
        delivery_id (str): Delivery id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AttemptsOutputBody | ErrorModel
    """

    return sync_detailed(
        delivery_id=delivery_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    delivery_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[AttemptsOutputBody | ErrorModel]:
    """Get a delivery's per-attempt history (any consumer)

     Same as getDeliveryAttempts, without a consumer.

    Portal tokens: not allowed.

    Args:
        delivery_id (str): Delivery id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AttemptsOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs(
        delivery_id=delivery_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    delivery_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> AttemptsOutputBody | ErrorModel | None:
    """Get a delivery's per-attempt history (any consumer)

     Same as getDeliveryAttempts, without a consumer.

    Portal tokens: not allowed.

    Args:
        delivery_id (str): Delivery id (UUID).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AttemptsOutputBody | ErrorModel
    """

    return (
        await asyncio_detailed(
            delivery_id=delivery_id,
            client=client,
        )
    ).parsed
