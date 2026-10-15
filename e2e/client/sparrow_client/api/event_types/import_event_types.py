from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.import_event_types_input_body import ImportEventTypesInputBody
from ...models.import_event_types_output_body import ImportEventTypesOutputBody
from ...types import Response


def _get_kwargs(
    *,
    body: ImportEventTypesInputBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/event-types:import",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | ImportEventTypesOutputBody | None:
    if response.status_code == 200:
        response_200 = ImportEventTypesOutputBody.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = ErrorModel.from_dict(response.json())

        return response_400

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
) -> Response[ErrorModel | ImportEventTypesOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: ImportEventTypesInputBody,
) -> Response[ErrorModel | ImportEventTypesOutputBody]:
    """Import a bundle of event type definitions

     Applies a bundle in one transaction: each item replaces the named event type using the usual version
    rules, or nothing is written. Event types not in the bundle are never touched; an import does not
    delete. The full per-item result is returned either way. Nothing is written for a dry run, when a
    stamp warning is not acknowledged, or when a change is breaking for subscriptions and allow_breaking
    is not set; blocked_by says why. The body may be an exported bundle as-is, plus the options.

    Args:
        body (ImportEventTypesInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ImportEventTypesOutputBody]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    body: ImportEventTypesInputBody,
) -> ErrorModel | ImportEventTypesOutputBody | None:
    """Import a bundle of event type definitions

     Applies a bundle in one transaction: each item replaces the named event type using the usual version
    rules, or nothing is written. Event types not in the bundle are never touched; an import does not
    delete. The full per-item result is returned either way. Nothing is written for a dry run, when a
    stamp warning is not acknowledged, or when a change is breaking for subscriptions and allow_breaking
    is not set; blocked_by says why. The body may be an exported bundle as-is, plus the options.

    Args:
        body (ImportEventTypesInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ImportEventTypesOutputBody
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: ImportEventTypesInputBody,
) -> Response[ErrorModel | ImportEventTypesOutputBody]:
    """Import a bundle of event type definitions

     Applies a bundle in one transaction: each item replaces the named event type using the usual version
    rules, or nothing is written. Event types not in the bundle are never touched; an import does not
    delete. The full per-item result is returned either way. Nothing is written for a dry run, when a
    stamp warning is not acknowledged, or when a change is breaking for subscriptions and allow_breaking
    is not set; blocked_by says why. The body may be an exported bundle as-is, plus the options.

    Args:
        body (ImportEventTypesInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | ImportEventTypesOutputBody]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: ImportEventTypesInputBody,
) -> ErrorModel | ImportEventTypesOutputBody | None:
    """Import a bundle of event type definitions

     Applies a bundle in one transaction: each item replaces the named event type using the usual version
    rules, or nothing is written. Event types not in the bundle are never touched; an import does not
    delete. The full per-item result is returned either way. Nothing is written for a dry run, when a
    stamp warning is not acknowledged, or when a change is breaking for subscriptions and allow_breaking
    is not set; blocked_by says why. The body may be an exported bundle as-is, plus the options.

    Args:
        body (ImportEventTypesInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | ImportEventTypesOutputBody
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
