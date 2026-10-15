from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.alert_config_item import AlertConfigItem
from ...models.create_alert_config_body import CreateAlertConfigBody
from ...models.error_model import ErrorModel
from ...types import Response


def _get_kwargs(
    consumer: str,
    *,
    body: CreateAlertConfigBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/consumers/{consumer}/alert-configs".format(
            consumer=quote(str(consumer), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> AlertConfigItem | ErrorModel | None:
    if response.status_code == 201:
        response_201 = AlertConfigItem.from_dict(response.json())

        return response_201

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
) -> Response[AlertConfigItem | ErrorModel]:
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
    body: CreateAlertConfigBody,
) -> Response[AlertConfigItem | ErrorModel]:
    """Register an email alert recipient

     Opts an email address into Sparrow's self-generated health-change, permanent-delivery-failure and
    auto-disable notifications, scoped to one webhook or the whole consumer. Emails are only sent once
    an alert mailer webhook is registered under the _sparrow consumer (see GET /v1/capabilities
    alert_delivery.configured). Not idempotent: each call adds a config.

    400 for an invalid email or event type, 404 when webhook_id is not in this consumer.

    Portal tokens: allowed, as POST /portal/api/alert-configs.

    Args:
        consumer (str): Consumer whose webhooks the alert covers.
        body (CreateAlertConfigBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AlertConfigItem | ErrorModel]
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
    body: CreateAlertConfigBody,
) -> AlertConfigItem | ErrorModel | None:
    """Register an email alert recipient

     Opts an email address into Sparrow's self-generated health-change, permanent-delivery-failure and
    auto-disable notifications, scoped to one webhook or the whole consumer. Emails are only sent once
    an alert mailer webhook is registered under the _sparrow consumer (see GET /v1/capabilities
    alert_delivery.configured). Not idempotent: each call adds a config.

    400 for an invalid email or event type, 404 when webhook_id is not in this consumer.

    Portal tokens: allowed, as POST /portal/api/alert-configs.

    Args:
        consumer (str): Consumer whose webhooks the alert covers.
        body (CreateAlertConfigBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AlertConfigItem | ErrorModel
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
    body: CreateAlertConfigBody,
) -> Response[AlertConfigItem | ErrorModel]:
    """Register an email alert recipient

     Opts an email address into Sparrow's self-generated health-change, permanent-delivery-failure and
    auto-disable notifications, scoped to one webhook or the whole consumer. Emails are only sent once
    an alert mailer webhook is registered under the _sparrow consumer (see GET /v1/capabilities
    alert_delivery.configured). Not idempotent: each call adds a config.

    400 for an invalid email or event type, 404 when webhook_id is not in this consumer.

    Portal tokens: allowed, as POST /portal/api/alert-configs.

    Args:
        consumer (str): Consumer whose webhooks the alert covers.
        body (CreateAlertConfigBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AlertConfigItem | ErrorModel]
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
    body: CreateAlertConfigBody,
) -> AlertConfigItem | ErrorModel | None:
    """Register an email alert recipient

     Opts an email address into Sparrow's self-generated health-change, permanent-delivery-failure and
    auto-disable notifications, scoped to one webhook or the whole consumer. Emails are only sent once
    an alert mailer webhook is registered under the _sparrow consumer (see GET /v1/capabilities
    alert_delivery.configured). Not idempotent: each call adds a config.

    400 for an invalid email or event type, 404 when webhook_id is not in this consumer.

    Portal tokens: allowed, as POST /portal/api/alert-configs.

    Args:
        consumer (str): Consumer whose webhooks the alert covers.
        body (CreateAlertConfigBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AlertConfigItem | ErrorModel
    """

    return (
        await asyncio_detailed(
            consumer=consumer,
            client=client,
            body=body,
        )
    ).parsed
