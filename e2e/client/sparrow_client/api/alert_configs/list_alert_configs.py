from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_alert_configs_output_body import ListAlertConfigsOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    consumer: str,
    *,
    webhook_id: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["webhook_id"] = webhook_id

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/consumers/{consumer}/alert-configs".format(
            consumer=quote(str(consumer), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ListAlertConfigsOutputBody:
    if response.status_code == 200:
        response_200 = ListAlertConfigsOutputBody.from_dict(response.json())

        return response_200

    response_default = ErrorModel.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorModel | ListAlertConfigsOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
) -> Response[ErrorModel | ListAlertConfigsOutputBody]:
    """List email alert recipients

     Lists alert configs in a consumer, oldest first, optionally filtered to one webhook's own configs.
    Not paginated.

    Portal tokens: allowed, as GET /portal/api/alert-configs.

    Args:
        consumer (str): Consumer to list alert configs in.
        webhook_id (str | Unset): Filter to alert configs scoped to one webhook. Omit to list
            every config in the consumer, including consumer-wide ones.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListAlertConfigsOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
) -> ErrorModel | ListAlertConfigsOutputBody | None:
    """List email alert recipients

     Lists alert configs in a consumer, oldest first, optionally filtered to one webhook's own configs.
    Not paginated.

    Portal tokens: allowed, as GET /portal/api/alert-configs.

    Args:
        consumer (str): Consumer to list alert configs in.
        webhook_id (str | Unset): Filter to alert configs scoped to one webhook. Omit to list
            every config in the consumer, including consumer-wide ones.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListAlertConfigsOutputBody
    """

    return sync_detailed(
        consumer=consumer,
        client=client,
        webhook_id=webhook_id,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
) -> Response[ErrorModel | ListAlertConfigsOutputBody]:
    """List email alert recipients

     Lists alert configs in a consumer, oldest first, optionally filtered to one webhook's own configs.
    Not paginated.

    Portal tokens: allowed, as GET /portal/api/alert-configs.

    Args:
        consumer (str): Consumer to list alert configs in.
        webhook_id (str | Unset): Filter to alert configs scoped to one webhook. Omit to list
            every config in the consumer, including consumer-wide ones.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListAlertConfigsOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    webhook_id: str | Unset = UNSET,
) -> ErrorModel | ListAlertConfigsOutputBody | None:
    """List email alert recipients

     Lists alert configs in a consumer, oldest first, optionally filtered to one webhook's own configs.
    Not paginated.

    Portal tokens: allowed, as GET /portal/api/alert-configs.

    Args:
        consumer (str): Consumer to list alert configs in.
        webhook_id (str | Unset): Filter to alert configs scoped to one webhook. Omit to list
            every config in the consumer, including consumer-wide ones.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListAlertConfigsOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            client=client,
            webhook_id=webhook_id,
        )
    ).parsed
