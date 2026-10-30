from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.capabilities_output_body import CapabilitiesOutputBody
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs() -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/capabilities",
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> CapabilitiesOutputBody | ErrorModel:
    if response.status_code == 200:
        response_200 = CapabilitiesOutputBody.from_dict(response.json())

        return response_200

    response_default = ErrorModel.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[CapabilitiesOutputBody | ErrorModel]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[CapabilitiesOutputBody | ErrorModel]:
    """List optional server features

     Reports which optional, deployment-configured features this server offers so clients can show or
    hide the matching UI: AI-assisted transform template drafting, with the provider and model in use,
    whether alert email delivery is set up, and whether portal tokens may retry deliveries. Read-only.

    ai_drafting.enabled reflects SPARROW_AI_ENABLED=true; drafting is usable only when enabled is true
    and problem is absent; prompt_only is the opposite of enabled.

    Portal tokens: allowed, as GET /portal/api/capabilities.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CapabilitiesOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs()

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
) -> CapabilitiesOutputBody | ErrorModel | None:
    """List optional server features

     Reports which optional, deployment-configured features this server offers so clients can show or
    hide the matching UI: AI-assisted transform template drafting, with the provider and model in use,
    whether alert email delivery is set up, and whether portal tokens may retry deliveries. Read-only.

    ai_drafting.enabled reflects SPARROW_AI_ENABLED=true; drafting is usable only when enabled is true
    and problem is absent; prompt_only is the opposite of enabled.

    Portal tokens: allowed, as GET /portal/api/capabilities.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CapabilitiesOutputBody | ErrorModel
    """

    return sync_detailed(
        client=client,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[CapabilitiesOutputBody | ErrorModel]:
    """List optional server features

     Reports which optional, deployment-configured features this server offers so clients can show or
    hide the matching UI: AI-assisted transform template drafting, with the provider and model in use,
    whether alert email delivery is set up, and whether portal tokens may retry deliveries. Read-only.

    ai_drafting.enabled reflects SPARROW_AI_ENABLED=true; drafting is usable only when enabled is true
    and problem is absent; prompt_only is the opposite of enabled.

    Portal tokens: allowed, as GET /portal/api/capabilities.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CapabilitiesOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs()

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
) -> CapabilitiesOutputBody | ErrorModel | None:
    """List optional server features

     Reports which optional, deployment-configured features this server offers so clients can show or
    hide the matching UI: AI-assisted transform template drafting, with the provider and model in use,
    whether alert email delivery is set up, and whether portal tokens may retry deliveries. Read-only.

    ai_drafting.enabled reflects SPARROW_AI_ENABLED=true; drafting is usable only when enabled is true
    and problem is absent; prompt_only is the opposite of enabled.

    Portal tokens: allowed, as GET /portal/api/capabilities.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CapabilitiesOutputBody | ErrorModel
    """

    return (
        await asyncio_detailed(
            client=client,
        )
    ).parsed
