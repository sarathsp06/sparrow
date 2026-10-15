from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.draft_template_body import DraftTemplateBody
from ...models.draft_template_output_body import DraftTemplateOutputBody
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs(
    *,
    body: DraftTemplateBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/subscriptions:draftTemplate",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> DraftTemplateOutputBody | ErrorModel | None:
    if response.status_code == 200:
        response_200 = DraftTemplateOutputBody.from_dict(response.json())

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

    if response.status_code == 429:
        response_429 = ErrorModel.from_dict(response.json())

        return response_429

    if response.status_code == 500:
        response_500 = ErrorModel.from_dict(response.json())

        return response_500

    if response.status_code == 503:
        response_503 = ErrorModel.from_dict(response.json())

        return response_503

    if response.status_code == 504:
        response_504 = ErrorModel.from_dict(response.json())

        return response_504

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[DraftTemplateOutputBody | ErrorModel]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: DraftTemplateBody,
) -> Response[DraftTemplateOutputBody | ErrorModel]:
    """Draft a transform template with AI

     Drafts a transform_template from a plain-language description, grounded in the event type's JSON
    Schema and sample payload, the template helper catalog, and optionally a shipped recipe's
    destination format or an example body the receiver expects. Every draft is rendered against the
    sample payload (as POST /v1/subscriptions:testTemplate does) and repaired until it renders, so the
    returned template is known to work. Nothing is saved: put the template into a subscription's
    transform_template. Requires SPARROW_AI_ENABLED=true and the provider settings (SPARROW_AI_API_KEY
    for Anthropic, or SPARROW_AI_PROVIDER=openai with SPARROW_AI_BASE_URL and SPARROW_AI_MODEL for any
    OpenAI-compatible server); otherwise 503, naming the missing setting. Only the sample payload
    (registered or provided in the request), the schema, and the request's own text are sent to the
    model, never stored events, headers, or secrets.

    Not idempotent (model output varies). 404 for an unknown event type or recipe, 429 when the provider
    rate-limits, 504 when the draft takes longer than SPARROW_AI_TIMEOUT (default 3m).

    Portal tokens: not allowed.

    Args:
        body (DraftTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DraftTemplateOutputBody | ErrorModel]
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
    body: DraftTemplateBody,
) -> DraftTemplateOutputBody | ErrorModel | None:
    """Draft a transform template with AI

     Drafts a transform_template from a plain-language description, grounded in the event type's JSON
    Schema and sample payload, the template helper catalog, and optionally a shipped recipe's
    destination format or an example body the receiver expects. Every draft is rendered against the
    sample payload (as POST /v1/subscriptions:testTemplate does) and repaired until it renders, so the
    returned template is known to work. Nothing is saved: put the template into a subscription's
    transform_template. Requires SPARROW_AI_ENABLED=true and the provider settings (SPARROW_AI_API_KEY
    for Anthropic, or SPARROW_AI_PROVIDER=openai with SPARROW_AI_BASE_URL and SPARROW_AI_MODEL for any
    OpenAI-compatible server); otherwise 503, naming the missing setting. Only the sample payload
    (registered or provided in the request), the schema, and the request's own text are sent to the
    model, never stored events, headers, or secrets.

    Not idempotent (model output varies). 404 for an unknown event type or recipe, 429 when the provider
    rate-limits, 504 when the draft takes longer than SPARROW_AI_TIMEOUT (default 3m).

    Portal tokens: not allowed.

    Args:
        body (DraftTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DraftTemplateOutputBody | ErrorModel
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: DraftTemplateBody,
) -> Response[DraftTemplateOutputBody | ErrorModel]:
    """Draft a transform template with AI

     Drafts a transform_template from a plain-language description, grounded in the event type's JSON
    Schema and sample payload, the template helper catalog, and optionally a shipped recipe's
    destination format or an example body the receiver expects. Every draft is rendered against the
    sample payload (as POST /v1/subscriptions:testTemplate does) and repaired until it renders, so the
    returned template is known to work. Nothing is saved: put the template into a subscription's
    transform_template. Requires SPARROW_AI_ENABLED=true and the provider settings (SPARROW_AI_API_KEY
    for Anthropic, or SPARROW_AI_PROVIDER=openai with SPARROW_AI_BASE_URL and SPARROW_AI_MODEL for any
    OpenAI-compatible server); otherwise 503, naming the missing setting. Only the sample payload
    (registered or provided in the request), the schema, and the request's own text are sent to the
    model, never stored events, headers, or secrets.

    Not idempotent (model output varies). 404 for an unknown event type or recipe, 429 when the provider
    rate-limits, 504 when the draft takes longer than SPARROW_AI_TIMEOUT (default 3m).

    Portal tokens: not allowed.

    Args:
        body (DraftTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DraftTemplateOutputBody | ErrorModel]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: DraftTemplateBody,
) -> DraftTemplateOutputBody | ErrorModel | None:
    """Draft a transform template with AI

     Drafts a transform_template from a plain-language description, grounded in the event type's JSON
    Schema and sample payload, the template helper catalog, and optionally a shipped recipe's
    destination format or an example body the receiver expects. Every draft is rendered against the
    sample payload (as POST /v1/subscriptions:testTemplate does) and repaired until it renders, so the
    returned template is known to work. Nothing is saved: put the template into a subscription's
    transform_template. Requires SPARROW_AI_ENABLED=true and the provider settings (SPARROW_AI_API_KEY
    for Anthropic, or SPARROW_AI_PROVIDER=openai with SPARROW_AI_BASE_URL and SPARROW_AI_MODEL for any
    OpenAI-compatible server); otherwise 503, naming the missing setting. Only the sample payload
    (registered or provided in the request), the schema, and the request's own text are sent to the
    model, never stored events, headers, or secrets.

    Not idempotent (model output varies). 404 for an unknown event type or recipe, 429 when the provider
    rate-limits, 504 when the draft takes longer than SPARROW_AI_TIMEOUT (default 3m).

    Portal tokens: not allowed.

    Args:
        body (DraftTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DraftTemplateOutputBody | ErrorModel
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
