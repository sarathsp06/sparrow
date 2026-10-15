from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.event_type_body import EventTypeBody
from ...models.update_event_type_output_body import UpdateEventTypeOutputBody
from ...types import UNSET, Response, Unset


def _get_kwargs(
    name: str,
    *,
    body: EventTypeBody,
    allow_breaking: bool | Unset = False,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    params: dict[str, Any] = {}

    params["allow_breaking"] = allow_breaking

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/v1/event-types/{name}".format(
            name=quote(str(name), safe=""),
        ),
        "params": params,
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | UpdateEventTypeOutputBody | None:
    if response.status_code == 200:
        response_200 = UpdateEventTypeOutputBody.from_dict(response.json())

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
) -> Response[ErrorModel | UpdateEventTypeOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: EventTypeBody,
    allow_breaking: bool | Unset = False,
) -> Response[ErrorModel | UpdateEventTypeOutputBody]:
    """Update an event type definition

     Merge-patches an event type: only fields present in the request body are changed. A schema change
    creates a new version and keeps the previous one; adding a first schema to a schema-less version,
    and any change to description, metadata or active, updates the current version in place. The
    response's change field says which happened. A breaking schema change (see change.compatibility) to
    an event type that subscriptions receive is refused with 409 unless allow_breaking=true.

    Args:
        name (str):
        allow_breaking (bool | Unset): Apply a breaking schema change even though subscriptions
            receive this event type. Without it, a breaking change to a subscribed type is refused
            with 409 and the reasons. Default: False.
        body (EventTypeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | UpdateEventTypeOutputBody]
    """

    kwargs = _get_kwargs(
        name=name,
        body=body,
        allow_breaking=allow_breaking,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: EventTypeBody,
    allow_breaking: bool | Unset = False,
) -> ErrorModel | UpdateEventTypeOutputBody | None:
    """Update an event type definition

     Merge-patches an event type: only fields present in the request body are changed. A schema change
    creates a new version and keeps the previous one; adding a first schema to a schema-less version,
    and any change to description, metadata or active, updates the current version in place. The
    response's change field says which happened. A breaking schema change (see change.compatibility) to
    an event type that subscriptions receive is refused with 409 unless allow_breaking=true.

    Args:
        name (str):
        allow_breaking (bool | Unset): Apply a breaking schema change even though subscriptions
            receive this event type. Without it, a breaking change to a subscribed type is refused
            with 409 and the reasons. Default: False.
        body (EventTypeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | UpdateEventTypeOutputBody
    """

    return sync_detailed(
        name=name,
        client=client,
        body=body,
        allow_breaking=allow_breaking,
    ).parsed


async def asyncio_detailed(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: EventTypeBody,
    allow_breaking: bool | Unset = False,
) -> Response[ErrorModel | UpdateEventTypeOutputBody]:
    """Update an event type definition

     Merge-patches an event type: only fields present in the request body are changed. A schema change
    creates a new version and keeps the previous one; adding a first schema to a schema-less version,
    and any change to description, metadata or active, updates the current version in place. The
    response's change field says which happened. A breaking schema change (see change.compatibility) to
    an event type that subscriptions receive is refused with 409 unless allow_breaking=true.

    Args:
        name (str):
        allow_breaking (bool | Unset): Apply a breaking schema change even though subscriptions
            receive this event type. Without it, a breaking change to a subscribed type is refused
            with 409 and the reasons. Default: False.
        body (EventTypeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | UpdateEventTypeOutputBody]
    """

    kwargs = _get_kwargs(
        name=name,
        body=body,
        allow_breaking=allow_breaking,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: EventTypeBody,
    allow_breaking: bool | Unset = False,
) -> ErrorModel | UpdateEventTypeOutputBody | None:
    """Update an event type definition

     Merge-patches an event type: only fields present in the request body are changed. A schema change
    creates a new version and keeps the previous one; adding a first schema to a schema-less version,
    and any change to description, metadata or active, updates the current version in place. The
    response's change field says which happened. A breaking schema change (see change.compatibility) to
    an event type that subscriptions receive is refused with 409 unless allow_breaking=true.

    Args:
        name (str):
        allow_breaking (bool | Unset): Apply a breaking schema change even though subscriptions
            receive this event type. Without it, a breaking change to a subscribed type is refused
            with 409 and the reasons. Default: False.
        body (EventTypeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | UpdateEventTypeOutputBody
    """

    return (
        await asyncio_detailed(
            name=name,
            client=client,
            body=body,
            allow_breaking=allow_breaking,
        )
    ).parsed
