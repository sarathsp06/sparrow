from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_deliveries_global_status import ListDeliveriesGlobalStatus
from ...models.list_deliveries_output_body import ListDeliveriesOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    consumer: str | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    event_id: str | Unset = UNSET,
    status: ListDeliveriesGlobalStatus | Unset = UNSET,
    error_category: str | Unset = UNSET,
    subscription_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    created_after: str | Unset = UNSET,
    created_before: str | Unset = UNSET,
    prepare_retry: bool | Unset = False,
    limit: int | Unset = 50,
    cursor: str | Unset = UNSET,
    offset: int | Unset = 0,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["consumer"] = consumer

    params["webhook_id"] = webhook_id

    params["event_id"] = event_id

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params["error_category"] = error_category

    params["subscription_id"] = subscription_id

    params["event_name"] = event_name

    params["created_after"] = created_after

    params["created_before"] = created_before

    params["prepare_retry"] = prepare_retry

    params["limit"] = limit

    params["cursor"] = cursor

    params["offset"] = offset

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/deliveries",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ListDeliveriesOutputBody:
    if response.status_code == 200:
        response_200 = ListDeliveriesOutputBody.from_dict(response.json())

        return response_200

    response_default = ErrorModel.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorModel | ListDeliveriesOutputBody]:
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
    webhook_id: str | Unset = UNSET,
    event_id: str | Unset = UNSET,
    status: ListDeliveriesGlobalStatus | Unset = UNSET,
    error_category: str | Unset = UNSET,
    subscription_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    created_after: str | Unset = UNSET,
    created_before: str | Unset = UNSET,
    prepare_retry: bool | Unset = False,
    limit: int | Unset = 50,
    cursor: str | Unset = UNSET,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListDeliveriesOutputBody]:
    """List deliveries across all consumers

     Cross-consumer delivery listing with the same filters as the per-consumer route. Pass consumer to
    scope to one consumer.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list deliveries across all
            consumers.
        webhook_id (str | Unset): Filter to deliveries for one webhook.
        event_id (str | Unset): Filter to deliveries for one pushed event occurrence.
        status (ListDeliveriesGlobalStatus | Unset): Filter by delivery status.
        error_category (str | Unset): Filter by failure classification (e.g. server_error,
            client_error, timeout).
        subscription_id (str | Unset): Filter to deliveries created by one subscription, e.g. its
            paused deliveries.
        event_name (str | Unset): Filter to deliveries of one event type, e.g. order.created. Also
            matches deliveries whose subscription has since been replaced or deleted.
        created_after (str | Unset): Filter to deliveries created on or after this date (YYYY-MM-
            DD) or exact time (RFC3339, e.g. an import's imported_at).
        created_before (str | Unset): Filter to deliveries created on or before this date (YYYY-
            MM-DD) or exact time (RFC3339).
        prepare_retry (bool | Unset): If true, snapshot the matching deliveries into a retry_id
            you can pass to the batch retry endpoint. Default: False.
        limit (int | Unset): Maximum items to return. Default: 50.
        cursor (str | Unset): Continue after the page this came from: pagination.next_cursor of
            the previous response. Stable while new deliveries arrive.
        offset (int | Unset): Removed: page with cursor. Any value other than 0 is rejected with
            400, so a client still paging by offset fails loudly instead of re-reading the first page.
            Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListDeliveriesOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
        event_id=event_id,
        status=status,
        error_category=error_category,
        subscription_id=subscription_id,
        event_name=event_name,
        created_after=created_after,
        created_before=created_before,
        prepare_retry=prepare_retry,
        limit=limit,
        cursor=cursor,
        offset=offset,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    event_id: str | Unset = UNSET,
    status: ListDeliveriesGlobalStatus | Unset = UNSET,
    error_category: str | Unset = UNSET,
    subscription_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    created_after: str | Unset = UNSET,
    created_before: str | Unset = UNSET,
    prepare_retry: bool | Unset = False,
    limit: int | Unset = 50,
    cursor: str | Unset = UNSET,
    offset: int | Unset = 0,
) -> ErrorModel | ListDeliveriesOutputBody | None:
    """List deliveries across all consumers

     Cross-consumer delivery listing with the same filters as the per-consumer route. Pass consumer to
    scope to one consumer.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list deliveries across all
            consumers.
        webhook_id (str | Unset): Filter to deliveries for one webhook.
        event_id (str | Unset): Filter to deliveries for one pushed event occurrence.
        status (ListDeliveriesGlobalStatus | Unset): Filter by delivery status.
        error_category (str | Unset): Filter by failure classification (e.g. server_error,
            client_error, timeout).
        subscription_id (str | Unset): Filter to deliveries created by one subscription, e.g. its
            paused deliveries.
        event_name (str | Unset): Filter to deliveries of one event type, e.g. order.created. Also
            matches deliveries whose subscription has since been replaced or deleted.
        created_after (str | Unset): Filter to deliveries created on or after this date (YYYY-MM-
            DD) or exact time (RFC3339, e.g. an import's imported_at).
        created_before (str | Unset): Filter to deliveries created on or before this date (YYYY-
            MM-DD) or exact time (RFC3339).
        prepare_retry (bool | Unset): If true, snapshot the matching deliveries into a retry_id
            you can pass to the batch retry endpoint. Default: False.
        limit (int | Unset): Maximum items to return. Default: 50.
        cursor (str | Unset): Continue after the page this came from: pagination.next_cursor of
            the previous response. Stable while new deliveries arrive.
        offset (int | Unset): Removed: page with cursor. Any value other than 0 is rejected with
            400, so a client still paging by offset fails loudly instead of re-reading the first page.
            Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListDeliveriesOutputBody
    """

    return sync_detailed(
        client=client,
        consumer=consumer,
        webhook_id=webhook_id,
        event_id=event_id,
        status=status,
        error_category=error_category,
        subscription_id=subscription_id,
        event_name=event_name,
        created_after=created_after,
        created_before=created_before,
        prepare_retry=prepare_retry,
        limit=limit,
        cursor=cursor,
        offset=offset,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    event_id: str | Unset = UNSET,
    status: ListDeliveriesGlobalStatus | Unset = UNSET,
    error_category: str | Unset = UNSET,
    subscription_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    created_after: str | Unset = UNSET,
    created_before: str | Unset = UNSET,
    prepare_retry: bool | Unset = False,
    limit: int | Unset = 50,
    cursor: str | Unset = UNSET,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListDeliveriesOutputBody]:
    """List deliveries across all consumers

     Cross-consumer delivery listing with the same filters as the per-consumer route. Pass consumer to
    scope to one consumer.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list deliveries across all
            consumers.
        webhook_id (str | Unset): Filter to deliveries for one webhook.
        event_id (str | Unset): Filter to deliveries for one pushed event occurrence.
        status (ListDeliveriesGlobalStatus | Unset): Filter by delivery status.
        error_category (str | Unset): Filter by failure classification (e.g. server_error,
            client_error, timeout).
        subscription_id (str | Unset): Filter to deliveries created by one subscription, e.g. its
            paused deliveries.
        event_name (str | Unset): Filter to deliveries of one event type, e.g. order.created. Also
            matches deliveries whose subscription has since been replaced or deleted.
        created_after (str | Unset): Filter to deliveries created on or after this date (YYYY-MM-
            DD) or exact time (RFC3339, e.g. an import's imported_at).
        created_before (str | Unset): Filter to deliveries created on or before this date (YYYY-
            MM-DD) or exact time (RFC3339).
        prepare_retry (bool | Unset): If true, snapshot the matching deliveries into a retry_id
            you can pass to the batch retry endpoint. Default: False.
        limit (int | Unset): Maximum items to return. Default: 50.
        cursor (str | Unset): Continue after the page this came from: pagination.next_cursor of
            the previous response. Stable while new deliveries arrive.
        offset (int | Unset): Removed: page with cursor. Any value other than 0 is rejected with
            400, so a client still paging by offset fails loudly instead of re-reading the first page.
            Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListDeliveriesOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        webhook_id=webhook_id,
        event_id=event_id,
        status=status,
        error_category=error_category,
        subscription_id=subscription_id,
        event_name=event_name,
        created_after=created_after,
        created_before=created_before,
        prepare_retry=prepare_retry,
        limit=limit,
        cursor=cursor,
        offset=offset,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    consumer: str | Unset = UNSET,
    webhook_id: str | Unset = UNSET,
    event_id: str | Unset = UNSET,
    status: ListDeliveriesGlobalStatus | Unset = UNSET,
    error_category: str | Unset = UNSET,
    subscription_id: str | Unset = UNSET,
    event_name: str | Unset = UNSET,
    created_after: str | Unset = UNSET,
    created_before: str | Unset = UNSET,
    prepare_retry: bool | Unset = False,
    limit: int | Unset = 50,
    cursor: str | Unset = UNSET,
    offset: int | Unset = 0,
) -> ErrorModel | ListDeliveriesOutputBody | None:
    """List deliveries across all consumers

     Cross-consumer delivery listing with the same filters as the per-consumer route. Pass consumer to
    scope to one consumer.

    Args:
        consumer (str | Unset): Filter to one consumer; omit to list deliveries across all
            consumers.
        webhook_id (str | Unset): Filter to deliveries for one webhook.
        event_id (str | Unset): Filter to deliveries for one pushed event occurrence.
        status (ListDeliveriesGlobalStatus | Unset): Filter by delivery status.
        error_category (str | Unset): Filter by failure classification (e.g. server_error,
            client_error, timeout).
        subscription_id (str | Unset): Filter to deliveries created by one subscription, e.g. its
            paused deliveries.
        event_name (str | Unset): Filter to deliveries of one event type, e.g. order.created. Also
            matches deliveries whose subscription has since been replaced or deleted.
        created_after (str | Unset): Filter to deliveries created on or after this date (YYYY-MM-
            DD) or exact time (RFC3339, e.g. an import's imported_at).
        created_before (str | Unset): Filter to deliveries created on or before this date (YYYY-
            MM-DD) or exact time (RFC3339).
        prepare_retry (bool | Unset): If true, snapshot the matching deliveries into a retry_id
            you can pass to the batch retry endpoint. Default: False.
        limit (int | Unset): Maximum items to return. Default: 50.
        cursor (str | Unset): Continue after the page this came from: pagination.next_cursor of
            the previous response. Stable while new deliveries arrive.
        offset (int | Unset): Removed: page with cursor. Any value other than 0 is rejected with
            400, so a client still paging by offset fails loudly instead of re-reading the first page.
            Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListDeliveriesOutputBody
    """

    return (
        await asyncio_detailed(
            client=client,
            consumer=consumer,
            webhook_id=webhook_id,
            event_id=event_id,
            status=status,
            error_category=error_category,
            subscription_id=subscription_id,
            event_name=event_name,
            created_after=created_after,
            created_before=created_before,
            prepare_retry=prepare_retry,
            limit=limit,
            cursor=cursor,
            offset=offset,
        )
    ).parsed
