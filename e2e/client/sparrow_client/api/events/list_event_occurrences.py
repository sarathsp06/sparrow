from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_event_occurrences_output_body import ListEventOccurrencesOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    consumer: str,
    *,
    event: str | Unset = UNSET,
    schema_valid: str | Unset = UNSET,
    labels: str | Unset = UNSET,
    created_after: str | Unset = UNSET,
    created_before: str | Unset = UNSET,
    prepare_repush: bool | Unset = False,
    limit: int | Unset = 50,
    cursor: str | Unset = UNSET,
    offset: int | Unset = 0,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["event"] = event

    params["schema_valid"] = schema_valid

    params["labels"] = labels

    params["created_after"] = created_after

    params["created_before"] = created_before

    params["prepare_repush"] = prepare_repush

    params["limit"] = limit

    params["cursor"] = cursor

    params["offset"] = offset

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/consumers/{consumer}/events".format(
            consumer=quote(str(consumer), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ListEventOccurrencesOutputBody:
    if response.status_code == 200:
        response_200 = ListEventOccurrencesOutputBody.from_dict(response.json())

        return response_200

    response_default = ErrorModel.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorModel | ListEventOccurrencesOutputBody]:
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
    event: str | Unset = UNSET,
    schema_valid: str | Unset = UNSET,
    labels: str | Unset = UNSET,
    created_after: str | Unset = UNSET,
    created_before: str | Unset = UNSET,
    prepare_repush: bool | Unset = False,
    limit: int | Unset = 50,
    cursor: str | Unset = UNSET,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListEventOccurrencesOutputBody]:
    """List pushed event occurrences

     Lists pushed event occurrences in a consumer, newest first, with delivery outcome counts per
    occurrence. Cursor paginated: pass pagination.next_cursor as cursor for the next page; there is no
    total count and offset is rejected.

    Side effect: prepare_repush=true also snapshots every match (up to 10,000) into a batch job and
    returns repush_id for POST .../events:rePush.

    Portal tokens: allowed, as GET /portal/api/events.

    Args:
        consumer (str): Consumer to list occurrences in.
        event (str | Unset): Filter to occurrences of this event type name.
        schema_valid (str | Unset): Filter by schema validation result: valid or invalid. Other
            values are 400.
        labels (str | Unset): Filter by labels as comma-separated key=value pairs, e.g.
            region=eu,tier=gold. Matches occurrences whose labels contain every pair.
        created_after (str | Unset): Only occurrences created at or after this time: YYYY-MM-DD
            (start of that day, UTC) or an RFC3339 timestamp.
        created_before (str | Unset): Only occurrences created at or before this time: YYYY-MM-DD
            (through the end of that day, UTC) or an RFC3339 timestamp.
        prepare_repush (bool | Unset): Side effect: when true, also snapshots the ids of every
            occurrence matching the filters (not just this page; at most 10,000, more is an error)
            into a batch job and returns its id as repush_id. Pass it to POST .../events:rePush within
            15 minutes. Each call creates a new snapshot. Default: False.
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        cursor (str | Unset): Continue after the page this came from: pagination.next_cursor of
            the previous response. Stable while new events arrive.
        offset (int | Unset): Removed: page with cursor. Any value other than 0 is rejected with
            400, so a client still paging by offset fails loudly instead of re-reading the first page.
            Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListEventOccurrencesOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        event=event,
        schema_valid=schema_valid,
        labels=labels,
        created_after=created_after,
        created_before=created_before,
        prepare_repush=prepare_repush,
        limit=limit,
        cursor=cursor,
        offset=offset,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    event: str | Unset = UNSET,
    schema_valid: str | Unset = UNSET,
    labels: str | Unset = UNSET,
    created_after: str | Unset = UNSET,
    created_before: str | Unset = UNSET,
    prepare_repush: bool | Unset = False,
    limit: int | Unset = 50,
    cursor: str | Unset = UNSET,
    offset: int | Unset = 0,
) -> ErrorModel | ListEventOccurrencesOutputBody | None:
    """List pushed event occurrences

     Lists pushed event occurrences in a consumer, newest first, with delivery outcome counts per
    occurrence. Cursor paginated: pass pagination.next_cursor as cursor for the next page; there is no
    total count and offset is rejected.

    Side effect: prepare_repush=true also snapshots every match (up to 10,000) into a batch job and
    returns repush_id for POST .../events:rePush.

    Portal tokens: allowed, as GET /portal/api/events.

    Args:
        consumer (str): Consumer to list occurrences in.
        event (str | Unset): Filter to occurrences of this event type name.
        schema_valid (str | Unset): Filter by schema validation result: valid or invalid. Other
            values are 400.
        labels (str | Unset): Filter by labels as comma-separated key=value pairs, e.g.
            region=eu,tier=gold. Matches occurrences whose labels contain every pair.
        created_after (str | Unset): Only occurrences created at or after this time: YYYY-MM-DD
            (start of that day, UTC) or an RFC3339 timestamp.
        created_before (str | Unset): Only occurrences created at or before this time: YYYY-MM-DD
            (through the end of that day, UTC) or an RFC3339 timestamp.
        prepare_repush (bool | Unset): Side effect: when true, also snapshots the ids of every
            occurrence matching the filters (not just this page; at most 10,000, more is an error)
            into a batch job and returns its id as repush_id. Pass it to POST .../events:rePush within
            15 minutes. Each call creates a new snapshot. Default: False.
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        cursor (str | Unset): Continue after the page this came from: pagination.next_cursor of
            the previous response. Stable while new events arrive.
        offset (int | Unset): Removed: page with cursor. Any value other than 0 is rejected with
            400, so a client still paging by offset fails loudly instead of re-reading the first page.
            Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListEventOccurrencesOutputBody
    """

    return sync_detailed(
        consumer=consumer,
        client=client,
        event=event,
        schema_valid=schema_valid,
        labels=labels,
        created_after=created_after,
        created_before=created_before,
        prepare_repush=prepare_repush,
        limit=limit,
        cursor=cursor,
        offset=offset,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    event: str | Unset = UNSET,
    schema_valid: str | Unset = UNSET,
    labels: str | Unset = UNSET,
    created_after: str | Unset = UNSET,
    created_before: str | Unset = UNSET,
    prepare_repush: bool | Unset = False,
    limit: int | Unset = 50,
    cursor: str | Unset = UNSET,
    offset: int | Unset = 0,
) -> Response[ErrorModel | ListEventOccurrencesOutputBody]:
    """List pushed event occurrences

     Lists pushed event occurrences in a consumer, newest first, with delivery outcome counts per
    occurrence. Cursor paginated: pass pagination.next_cursor as cursor for the next page; there is no
    total count and offset is rejected.

    Side effect: prepare_repush=true also snapshots every match (up to 10,000) into a batch job and
    returns repush_id for POST .../events:rePush.

    Portal tokens: allowed, as GET /portal/api/events.

    Args:
        consumer (str): Consumer to list occurrences in.
        event (str | Unset): Filter to occurrences of this event type name.
        schema_valid (str | Unset): Filter by schema validation result: valid or invalid. Other
            values are 400.
        labels (str | Unset): Filter by labels as comma-separated key=value pairs, e.g.
            region=eu,tier=gold. Matches occurrences whose labels contain every pair.
        created_after (str | Unset): Only occurrences created at or after this time: YYYY-MM-DD
            (start of that day, UTC) or an RFC3339 timestamp.
        created_before (str | Unset): Only occurrences created at or before this time: YYYY-MM-DD
            (through the end of that day, UTC) or an RFC3339 timestamp.
        prepare_repush (bool | Unset): Side effect: when true, also snapshots the ids of every
            occurrence matching the filters (not just this page; at most 10,000, more is an error)
            into a batch job and returns its id as repush_id. Pass it to POST .../events:rePush within
            15 minutes. Each call creates a new snapshot. Default: False.
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        cursor (str | Unset): Continue after the page this came from: pagination.next_cursor of
            the previous response. Stable while new events arrive.
        offset (int | Unset): Removed: page with cursor. Any value other than 0 is rejected with
            400, so a client still paging by offset fails loudly instead of re-reading the first page.
            Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListEventOccurrencesOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        event=event,
        schema_valid=schema_valid,
        labels=labels,
        created_after=created_after,
        created_before=created_before,
        prepare_repush=prepare_repush,
        limit=limit,
        cursor=cursor,
        offset=offset,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    event: str | Unset = UNSET,
    schema_valid: str | Unset = UNSET,
    labels: str | Unset = UNSET,
    created_after: str | Unset = UNSET,
    created_before: str | Unset = UNSET,
    prepare_repush: bool | Unset = False,
    limit: int | Unset = 50,
    cursor: str | Unset = UNSET,
    offset: int | Unset = 0,
) -> ErrorModel | ListEventOccurrencesOutputBody | None:
    """List pushed event occurrences

     Lists pushed event occurrences in a consumer, newest first, with delivery outcome counts per
    occurrence. Cursor paginated: pass pagination.next_cursor as cursor for the next page; there is no
    total count and offset is rejected.

    Side effect: prepare_repush=true also snapshots every match (up to 10,000) into a batch job and
    returns repush_id for POST .../events:rePush.

    Portal tokens: allowed, as GET /portal/api/events.

    Args:
        consumer (str): Consumer to list occurrences in.
        event (str | Unset): Filter to occurrences of this event type name.
        schema_valid (str | Unset): Filter by schema validation result: valid or invalid. Other
            values are 400.
        labels (str | Unset): Filter by labels as comma-separated key=value pairs, e.g.
            region=eu,tier=gold. Matches occurrences whose labels contain every pair.
        created_after (str | Unset): Only occurrences created at or after this time: YYYY-MM-DD
            (start of that day, UTC) or an RFC3339 timestamp.
        created_before (str | Unset): Only occurrences created at or before this time: YYYY-MM-DD
            (through the end of that day, UTC) or an RFC3339 timestamp.
        prepare_repush (bool | Unset): Side effect: when true, also snapshots the ids of every
            occurrence matching the filters (not just this page; at most 10,000, more is an error)
            into a batch job and returns its id as repush_id. Pass it to POST .../events:rePush within
            15 minutes. Each call creates a new snapshot. Default: False.
        limit (int | Unset): Maximum items to return, 1 to 1000. Default: 50.
        cursor (str | Unset): Continue after the page this came from: pagination.next_cursor of
            the previous response. Stable while new events arrive.
        offset (int | Unset): Removed: page with cursor. Any value other than 0 is rejected with
            400, so a client still paging by offset fails loudly instead of re-reading the first page.
            Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListEventOccurrencesOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            client=client,
            event=event,
            schema_valid=schema_valid,
            labels=labels,
            created_after=created_after,
            created_before=created_before,
            prepare_repush=prepare_repush,
            limit=limit,
            cursor=cursor,
            offset=offset,
        )
    ).parsed
