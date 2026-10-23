from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.batch_job_output_body import BatchJobOutputBody
from ...models.error_model import ErrorModel
from ...models.retry_batch_input_body import RetryBatchInputBody
from ...types import Response


def _get_kwargs(
    consumer: str,
    *,
    body: RetryBatchInputBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/consumers/{consumer}/deliveries:retryBatch".format(
            consumer=quote(str(consumer), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> BatchJobOutputBody | ErrorModel | None:
    if response.status_code == 202:
        response_202 = BatchJobOutputBody.from_dict(response.json())

        return response_202

    if response.status_code == 400:
        response_400 = ErrorModel.from_dict(response.json())

        return response_400

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

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[BatchJobOutputBody | ErrorModel]:
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
    body: RetryBatchInputBody,
) -> Response[BatchJobOutputBody | ErrorModel]:
    """Start a batch retry job from a prepared snapshot

     Starts an async job that retries every delivery in a snapshot taken by an earlier GET
    .../deliveries?prepare_retry=true call, like retryDelivery for each one. Send the retry_id the
    deliveries list returned. Returns 202 with the job; poll it with getDeliveryRetryJob.

    A snapshot can be started once: 409 when it is not pending (already started or cancelled), expired
    (15 minutes after it was taken), or is an event re-push snapshot; 404 when the id is unknown, 400
    when it is not a UUID.

    Portal tokens: allowed, as POST /portal/api/deliveries:retryBatch, unless the server runs with
    SPARROW_PORTAL_RETRY=false (then 403).

    Args:
        consumer (str): Consumer the job belongs to. A job snapshotted for another consumer
            answers 404; a job snapshotted across all consumers is reachable under any consumer,
            except with a portal token.
        body (RetryBatchInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BatchJobOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    body: RetryBatchInputBody,
) -> BatchJobOutputBody | ErrorModel | None:
    """Start a batch retry job from a prepared snapshot

     Starts an async job that retries every delivery in a snapshot taken by an earlier GET
    .../deliveries?prepare_retry=true call, like retryDelivery for each one. Send the retry_id the
    deliveries list returned. Returns 202 with the job; poll it with getDeliveryRetryJob.

    A snapshot can be started once: 409 when it is not pending (already started or cancelled), expired
    (15 minutes after it was taken), or is an event re-push snapshot; 404 when the id is unknown, 400
    when it is not a UUID.

    Portal tokens: allowed, as POST /portal/api/deliveries:retryBatch, unless the server runs with
    SPARROW_PORTAL_RETRY=false (then 403).

    Args:
        consumer (str): Consumer the job belongs to. A job snapshotted for another consumer
            answers 404; a job snapshotted across all consumers is reachable under any consumer,
            except with a portal token.
        body (RetryBatchInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BatchJobOutputBody | ErrorModel
    """

    return sync_detailed(
        consumer=consumer,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    body: RetryBatchInputBody,
) -> Response[BatchJobOutputBody | ErrorModel]:
    """Start a batch retry job from a prepared snapshot

     Starts an async job that retries every delivery in a snapshot taken by an earlier GET
    .../deliveries?prepare_retry=true call, like retryDelivery for each one. Send the retry_id the
    deliveries list returned. Returns 202 with the job; poll it with getDeliveryRetryJob.

    A snapshot can be started once: 409 when it is not pending (already started or cancelled), expired
    (15 minutes after it was taken), or is an event re-push snapshot; 404 when the id is unknown, 400
    when it is not a UUID.

    Portal tokens: allowed, as POST /portal/api/deliveries:retryBatch, unless the server runs with
    SPARROW_PORTAL_RETRY=false (then 403).

    Args:
        consumer (str): Consumer the job belongs to. A job snapshotted for another consumer
            answers 404; a job snapshotted across all consumers is reachable under any consumer,
            except with a portal token.
        body (RetryBatchInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BatchJobOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    *,
    client: AuthenticatedClient | Client,
    body: RetryBatchInputBody,
) -> BatchJobOutputBody | ErrorModel | None:
    """Start a batch retry job from a prepared snapshot

     Starts an async job that retries every delivery in a snapshot taken by an earlier GET
    .../deliveries?prepare_retry=true call, like retryDelivery for each one. Send the retry_id the
    deliveries list returned. Returns 202 with the job; poll it with getDeliveryRetryJob.

    A snapshot can be started once: 409 when it is not pending (already started or cancelled), expired
    (15 minutes after it was taken), or is an event re-push snapshot; 404 when the id is unknown, 400
    when it is not a UUID.

    Portal tokens: allowed, as POST /portal/api/deliveries:retryBatch, unless the server runs with
    SPARROW_PORTAL_RETRY=false (then 403).

    Args:
        consumer (str): Consumer the job belongs to. A job snapshotted for another consumer
            answers 404; a job snapshotted across all consumers is reachable under any consumer,
            except with a portal token.
        body (RetryBatchInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BatchJobOutputBody | ErrorModel
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            client=client,
            body=body,
        )
    ).parsed
