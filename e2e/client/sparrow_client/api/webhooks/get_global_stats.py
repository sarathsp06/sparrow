from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.consumer_stats_output_body import ConsumerStatsOutputBody
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs() -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/stats",
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ConsumerStatsOutputBody | ErrorModel:
    if response.status_code == 200:
        response_200 = ConsumerStatsOutputBody.from_dict(response.json())

        return response_200

    response_default = ErrorModel.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ConsumerStatsOutputBody | ErrorModel]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[ConsumerStatsOutputBody | ErrorModel]:
    """Get aggregate delivery statistics across all consumers

     Returns the same counters as the per-consumer stats endpoint, summed across every consumer, with the
    same lag.

    Portal tokens: not allowed.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConsumerStatsOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs()

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
) -> ConsumerStatsOutputBody | ErrorModel | None:
    """Get aggregate delivery statistics across all consumers

     Returns the same counters as the per-consumer stats endpoint, summed across every consumer, with the
    same lag.

    Portal tokens: not allowed.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConsumerStatsOutputBody | ErrorModel
    """

    return sync_detailed(
        client=client,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[ConsumerStatsOutputBody | ErrorModel]:
    """Get aggregate delivery statistics across all consumers

     Returns the same counters as the per-consumer stats endpoint, summed across every consumer, with the
    same lag.

    Portal tokens: not allowed.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConsumerStatsOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs()

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
) -> ConsumerStatsOutputBody | ErrorModel | None:
    """Get aggregate delivery statistics across all consumers

     Returns the same counters as the per-consumer stats endpoint, summed across every consumer, with the
    same lag.

    Portal tokens: not allowed.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConsumerStatsOutputBody | ErrorModel
    """

    return (
        await asyncio_detailed(
            client=client,
        )
    ).parsed
