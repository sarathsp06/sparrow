from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs(
    consumer: str,
    alert_config_id: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/v1/consumers/{consumer}/alert-configs/{alert_config_id}".format(
            consumer=quote(str(consumer), safe=""),
            alert_config_id=quote(str(alert_config_id), safe=""),
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
    consumer: str,
    alert_config_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | ErrorModel]:
    """Delete an email alert recipient

     Removes an alert config. The webhook and its delivery history are unaffected.

    Args:
        consumer (str):
        alert_config_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ErrorModel]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        alert_config_id=alert_config_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    alert_config_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Any | ErrorModel | None:
    """Delete an email alert recipient

     Removes an alert config. The webhook and its delivery history are unaffected.

    Args:
        consumer (str):
        alert_config_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ErrorModel
    """

    return sync_detailed(
        consumer=consumer,
        alert_config_id=alert_config_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    alert_config_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | ErrorModel]:
    """Delete an email alert recipient

     Removes an alert config. The webhook and its delivery history are unaffected.

    Args:
        consumer (str):
        alert_config_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ErrorModel]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        alert_config_id=alert_config_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    alert_config_id: str,
    *,
    client: AuthenticatedClient | Client,
) -> Any | ErrorModel | None:
    """Delete an email alert recipient

     Removes an alert config. The webhook and its delivery history are unaffected.

    Args:
        consumer (str):
        alert_config_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ErrorModel
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            alert_config_id=alert_config_id,
            client=client,
        )
    ).parsed
