from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.health_summary_output_body import HealthSummaryOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    consumer: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["consumer"] = consumer

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/health-summary",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | HealthSummaryOutputBody:
    if response.status_code == 200:
        response_200 = HealthSummaryOutputBody.from_dict(response.json())

        return response_200

    response_default = ErrorModel.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorModel | HealthSummaryOutputBody]:
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
) -> Response[ErrorModel | HealthSummaryOutputBody]:
    """Get aggregate webhook health counts

     Returns how many webhooks are currently healthy, degraded, unhealthy, or unknown, in one consumer or
    across every consumer — for a top-level dashboard tile, plus the thresholds behind those statuses.
    Deleted webhooks are not counted.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Count one consumer's webhooks; omit to count across all consumers.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | HealthSummaryOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
) -> ErrorModel | HealthSummaryOutputBody | None:
    """Get aggregate webhook health counts

     Returns how many webhooks are currently healthy, degraded, unhealthy, or unknown, in one consumer or
    across every consumer — for a top-level dashboard tile, plus the thresholds behind those statuses.
    Deleted webhooks are not counted.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Count one consumer's webhooks; omit to count across all consumers.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | HealthSummaryOutputBody
    """

    return sync_detailed(
        client=client,
        consumer=consumer,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
) -> Response[ErrorModel | HealthSummaryOutputBody]:
    """Get aggregate webhook health counts

     Returns how many webhooks are currently healthy, degraded, unhealthy, or unknown, in one consumer or
    across every consumer — for a top-level dashboard tile, plus the thresholds behind those statuses.
    Deleted webhooks are not counted.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Count one consumer's webhooks; omit to count across all consumers.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | HealthSummaryOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
) -> ErrorModel | HealthSummaryOutputBody | None:
    """Get aggregate webhook health counts

     Returns how many webhooks are currently healthy, degraded, unhealthy, or unknown, in one consumer or
    across every consumer — for a top-level dashboard tile, plus the thresholds behind those statuses.
    Deleted webhooks are not counted.

    Portal tokens: not allowed.

    Args:
        consumer (str | Unset): Count one consumer's webhooks; omit to count across all consumers.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | HealthSummaryOutputBody
    """

    return (
        await asyncio_detailed(
            client=client,
            consumer=consumer,
        )
    ).parsed
