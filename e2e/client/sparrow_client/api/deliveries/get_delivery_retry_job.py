from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.batch_job_output_body import BatchJobOutputBody
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs(
    consumer: str,
    job_id: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/consumers/{consumer}/retry-jobs/{job_id}".format(
            consumer=quote(str(consumer), safe=""),
            job_id=quote(str(job_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> BatchJobOutputBody | ErrorModel | None:
    if response.status_code == 200:
        response_200 = BatchJobOutputBody.from_dict(response.json())

        return response_200

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
    job_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[BatchJobOutputBody | ErrorModel]:
    """Get batch retry job progress

     Returns a batch retry job's status and processed/failed/total counts. 404 when the id is unknown,
    409 when the id belongs to an event re-push job.

    Portal tokens: allowed, as GET /portal/api/retry-jobs/{job_id}.

    Args:
        consumer (str): Consumer the job belongs to. A job snapshotted for another consumer
            answers 404; a job snapshotted across all consumers is reachable under any consumer,
            except with a portal token.
        job_id (str): Batch job id (UUID): the repush_id or retry_id from the list call.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BatchJobOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        job_id=job_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    job_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> BatchJobOutputBody | ErrorModel | None:
    """Get batch retry job progress

     Returns a batch retry job's status and processed/failed/total counts. 404 when the id is unknown,
    409 when the id belongs to an event re-push job.

    Portal tokens: allowed, as GET /portal/api/retry-jobs/{job_id}.

    Args:
        consumer (str): Consumer the job belongs to. A job snapshotted for another consumer
            answers 404; a job snapshotted across all consumers is reachable under any consumer,
            except with a portal token.
        job_id (str): Batch job id (UUID): the repush_id or retry_id from the list call.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BatchJobOutputBody | ErrorModel
    """

    return sync_detailed(
        consumer=consumer,
        job_id=job_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    job_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[BatchJobOutputBody | ErrorModel]:
    """Get batch retry job progress

     Returns a batch retry job's status and processed/failed/total counts. 404 when the id is unknown,
    409 when the id belongs to an event re-push job.

    Portal tokens: allowed, as GET /portal/api/retry-jobs/{job_id}.

    Args:
        consumer (str): Consumer the job belongs to. A job snapshotted for another consumer
            answers 404; a job snapshotted across all consumers is reachable under any consumer,
            except with a portal token.
        job_id (str): Batch job id (UUID): the repush_id or retry_id from the list call.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[BatchJobOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        job_id=job_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    job_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> BatchJobOutputBody | ErrorModel | None:
    """Get batch retry job progress

     Returns a batch retry job's status and processed/failed/total counts. 404 when the id is unknown,
    409 when the id belongs to an event re-push job.

    Portal tokens: allowed, as GET /portal/api/retry-jobs/{job_id}.

    Args:
        consumer (str): Consumer the job belongs to. A job snapshotted for another consumer
            answers 404; a job snapshotted across all consumers is reachable under any consumer,
            except with a portal token.
        job_id (str): Batch job id (UUID): the repush_id or retry_id from the list call.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        BatchJobOutputBody | ErrorModel
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            job_id=job_id,
            client=client,
        )
    ).parsed
