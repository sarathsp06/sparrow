from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error_model import ErrorModel
from ...models.test_template_body import TestTemplateBody
from ...models.test_template_output_body import TestTemplateOutputBody
from ...types import Response


def _get_kwargs(
    *,
    body: TestTemplateBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/subscriptions:testTemplate",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ErrorModel | TestTemplateOutputBody | None:
    if response.status_code == 200:
        response_200 = TestTemplateOutputBody.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = ErrorModel.from_dict(response.json())

        return response_400

    if response.status_code == 404:
        response_404 = ErrorModel.from_dict(response.json())

        return response_404

    if response.status_code == 422:
        response_422 = ErrorModel.from_dict(response.json())

        return response_422

    if response.status_code == 500:
        response_500 = ErrorModel.from_dict(response.json())

        return response_500

    if response.status_code == 503:
        response_503 = ErrorModel.from_dict(response.json())

        return response_503

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ErrorModel | TestTemplateOutputBody]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: TestTemplateBody,
) -> Response[ErrorModel | TestTemplateOutputBody]:
    """Render a transform template against an event's sample payload

     Dry-runs a transform template against the named event type's sample payload, without creating a
    subscription or any delivery. Use this to iterate on transform_template before saving it. Stateless
    and idempotent.

    400 when the template does not parse or fails to render (with template_missing_key error, a key the
    sample lacks is an error), 404 when the event type is not registered, 503 when the template engine
    is at capacity (retry).

    Portal tokens: allowed, as POST /portal/api/subscriptions:testTemplate.

    Args:
        body (TestTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | TestTemplateOutputBody]
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
    body: TestTemplateBody,
) -> ErrorModel | TestTemplateOutputBody | None:
    """Render a transform template against an event's sample payload

     Dry-runs a transform template against the named event type's sample payload, without creating a
    subscription or any delivery. Use this to iterate on transform_template before saving it. Stateless
    and idempotent.

    400 when the template does not parse or fails to render (with template_missing_key error, a key the
    sample lacks is an error), 404 when the event type is not registered, 503 when the template engine
    is at capacity (retry).

    Portal tokens: allowed, as POST /portal/api/subscriptions:testTemplate.

    Args:
        body (TestTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | TestTemplateOutputBody
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: TestTemplateBody,
) -> Response[ErrorModel | TestTemplateOutputBody]:
    """Render a transform template against an event's sample payload

     Dry-runs a transform template against the named event type's sample payload, without creating a
    subscription or any delivery. Use this to iterate on transform_template before saving it. Stateless
    and idempotent.

    400 when the template does not parse or fails to render (with template_missing_key error, a key the
    sample lacks is an error), 404 when the event type is not registered, 503 when the template engine
    is at capacity (retry).

    Portal tokens: allowed, as POST /portal/api/subscriptions:testTemplate.

    Args:
        body (TestTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ErrorModel | TestTemplateOutputBody]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: TestTemplateBody,
) -> ErrorModel | TestTemplateOutputBody | None:
    """Render a transform template against an event's sample payload

     Dry-runs a transform template against the named event type's sample payload, without creating a
    subscription or any delivery. Use this to iterate on transform_template before saving it. Stateless
    and idempotent.

    400 when the template does not parse or fails to render (with template_missing_key error, a key the
    sample lacks is an error), 404 when the event type is not registered, 503 when the template engine
    is at capacity (retry).

    Portal tokens: allowed, as POST /portal/api/subscriptions:testTemplate.

    Args:
        body (TestTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ErrorModel | TestTemplateOutputBody
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
