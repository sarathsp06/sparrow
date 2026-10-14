from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.list_template_versions_output_body import ListTemplateVersionsOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    consumer: str,
    subscription_id: str,
    *,
    limit: int | Unset = 20,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["limit"] = limit

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/consumers/{consumer}/subscriptions/{subscription_id}/templateVersions".format(
            consumer=quote(str(consumer), safe=""),
            subscription_id=quote(str(subscription_id), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ListTemplateVersionsOutputBody | None:
    if response.status_code == 200:
        response_200 = ListTemplateVersionsOutputBody.from_dict(response.json())

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
) -> Response[ErrorModel | ListTemplateVersionsOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    consumer: str,
    subscription_id: str,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = 20,
) -> Response[ErrorModel | ListTemplateVersionsOutputBody]:
    """List a subscription's saved template history

     Returns the transform templates this subscription has been saved with, newest first; the newest is
    the current one. A version is recorded whenever a save changes the template, with how it was
    produced (manual or ai_draft), an optional note, and who saved it. The last 20 are kept. There is no
    restore endpoint: load a version into the editor and save the subscription, which records it as a
    new version.

    Args:
        consumer (str):
        subscription_id (str):
        limit (int | Unset): Max versions to return, newest first. Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListTemplateVersionsOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        subscription_id=subscription_id,
        limit=limit,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    consumer: str,
    subscription_id: str,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = 20,
) -> ErrorModel | ListTemplateVersionsOutputBody | None:
    """List a subscription's saved template history

     Returns the transform templates this subscription has been saved with, newest first; the newest is
    the current one. A version is recorded whenever a save changes the template, with how it was
    produced (manual or ai_draft), an optional note, and who saved it. The last 20 are kept. There is no
    restore endpoint: load a version into the editor and save the subscription, which records it as a
    new version.

    Args:
        consumer (str):
        subscription_id (str):
        limit (int | Unset): Max versions to return, newest first. Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListTemplateVersionsOutputBody
    """

    return sync_detailed(
        consumer=consumer,
        subscription_id=subscription_id,
        client=client,
        limit=limit,
    ).parsed


async def asyncio_detailed(
    consumer: str,
    subscription_id: str,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = 20,
) -> Response[ErrorModel | ListTemplateVersionsOutputBody]:
    """List a subscription's saved template history

     Returns the transform templates this subscription has been saved with, newest first; the newest is
    the current one. A version is recorded whenever a save changes the template, with how it was
    produced (manual or ai_draft), an optional note, and who saved it. The last 20 are kept. There is no
    restore endpoint: load a version into the editor and save the subscription, which records it as a
    new version.

    Args:
        consumer (str):
        subscription_id (str):
        limit (int | Unset): Max versions to return, newest first. Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ListTemplateVersionsOutputBody]
    """

    kwargs = _get_kwargs(
        consumer=consumer,
        subscription_id=subscription_id,
        limit=limit,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    consumer: str,
    subscription_id: str,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = 20,
) -> ErrorModel | ListTemplateVersionsOutputBody | None:
    """List a subscription's saved template history

     Returns the transform templates this subscription has been saved with, newest first; the newest is
    the current one. A version is recorded whenever a save changes the template, with how it was
    produced (manual or ai_draft), an optional note, and who saved it. The last 20 are kept. There is no
    restore endpoint: load a version into the editor and save the subscription, which records it as a
    new version.

    Args:
        consumer (str):
        subscription_id (str):
        limit (int | Unset): Max versions to return, newest first. Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ListTemplateVersionsOutputBody
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            subscription_id=subscription_id,
            client=client,
            limit=limit,
        )
    ).parsed
