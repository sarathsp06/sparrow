from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.draft_template_body import DraftTemplateBody
from ...models.draft_template_prompt_output_body import DraftTemplatePromptOutputBody
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs(
    *,
    body: DraftTemplateBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/subscriptions:draftTemplatePrompt",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> DraftTemplatePromptOutputBody | ErrorModel | None:
    if response.status_code == 200:
        response_200 = DraftTemplatePromptOutputBody.from_dict(response.json())

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

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[DraftTemplatePromptOutputBody | ErrorModel]:
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
) -> Response[DraftTemplatePromptOutputBody | ErrorModel]:
    """Build the AI drafting prompt without calling a model

     Returns the same prompt POST /v1/subscriptions:draftTemplate would send, as one self-contained text
    to paste into any chat assistant (ChatGPT, Claude, a local model). Works whether or not the server
    has an AI provider configured, so installs without SPARROW_AI_* still get a one-click prompt. The
    user pastes the assistant's template back into the subscription editor and checks it with Run
    Preview. Nothing is sent anywhere by Sparrow; a docs_url, if given, is fetched under the delivery
    network policy to include an excerpt. No side effects.

    Portal tokens: not allowed.

    Args:
        body (DraftTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DraftTemplatePromptOutputBody | ErrorModel]
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
) -> DraftTemplatePromptOutputBody | ErrorModel | None:
    """Build the AI drafting prompt without calling a model

     Returns the same prompt POST /v1/subscriptions:draftTemplate would send, as one self-contained text
    to paste into any chat assistant (ChatGPT, Claude, a local model). Works whether or not the server
    has an AI provider configured, so installs without SPARROW_AI_* still get a one-click prompt. The
    user pastes the assistant's template back into the subscription editor and checks it with Run
    Preview. Nothing is sent anywhere by Sparrow; a docs_url, if given, is fetched under the delivery
    network policy to include an excerpt. No side effects.

    Portal tokens: not allowed.

    Args:
        body (DraftTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DraftTemplatePromptOutputBody | ErrorModel
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: DraftTemplateBody,
) -> Response[DraftTemplatePromptOutputBody | ErrorModel]:
    """Build the AI drafting prompt without calling a model

     Returns the same prompt POST /v1/subscriptions:draftTemplate would send, as one self-contained text
    to paste into any chat assistant (ChatGPT, Claude, a local model). Works whether or not the server
    has an AI provider configured, so installs without SPARROW_AI_* still get a one-click prompt. The
    user pastes the assistant's template back into the subscription editor and checks it with Run
    Preview. Nothing is sent anywhere by Sparrow; a docs_url, if given, is fetched under the delivery
    network policy to include an excerpt. No side effects.

    Portal tokens: not allowed.

    Args:
        body (DraftTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DraftTemplatePromptOutputBody | ErrorModel]
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
) -> DraftTemplatePromptOutputBody | ErrorModel | None:
    """Build the AI drafting prompt without calling a model

     Returns the same prompt POST /v1/subscriptions:draftTemplate would send, as one self-contained text
    to paste into any chat assistant (ChatGPT, Claude, a local model). Works whether or not the server
    has an AI provider configured, so installs without SPARROW_AI_* still get a one-click prompt. The
    user pastes the assistant's template back into the subscription editor and checks it with Run
    Preview. Nothing is sent anywhere by Sparrow; a docs_url, if given, is fetched under the delivery
    network policy to include an excerpt. No side effects.

    Portal tokens: not allowed.

    Args:
        body (DraftTemplateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DraftTemplatePromptOutputBody | ErrorModel
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
