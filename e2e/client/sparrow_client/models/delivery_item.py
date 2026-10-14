from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..models.delivery_item_error_category import DeliveryItemErrorCategory
from ..models.delivery_item_status import DeliveryItemStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="DeliveryItem")


@_attrs_define
class DeliveryItem:
    """
    Attributes:
        attempt_count (int): Number of delivery attempts made so far.
        created_at (str): Creation timestamp, RFC3339.
        delivery_id (str): Delivery id (UUID).
        event_id (str): Pushed event occurrence this delivery originated from.
        max_attempts (int): Maximum attempts allowed before the delivery is marked failed.
        status (DeliveryItemStatus): Current delivery status. paused: created while its subscription was paused; not
            attempted until retried.
        webhook_id (str): Webhook this delivery was sent to.
        schema (str | Unset): A URL to the JSON Schema for this object.
        consumer (str | Unset): Consumer of the webhook. Set in delivery listings.
        error_category (DeliveryItemErrorCategory | Unset): Failure classification from the most recent attempt, used to
            decide retryability. template_error means the subscription's transform failed to render; it is not retried
            automatically and does not count against the webhook's health.
        error_message (str | Unset): Human-readable failure reason from the most recent attempt.
        event_name (str | Unset): Event type of that occurrence, e.g. order.created. Set in delivery listings.
        last_attempted_at (str | Unset): Timestamp of the most recent attempt, RFC3339.
        next_retry_at (str | Unset): Timestamp of the next scheduled retry, RFC3339, if one is pending.
        response_body (str | Unset): Endpoint response body from the most recent attempt, if capture_response_body is
            enabled.
        response_code (int | Unset): HTTP status code returned by the endpoint on the most recent attempt, if any.
        template_error (str | Unset): The payload transform error, when the subscription's template failed to render.
            Set for both on_transform_error modes: with fail the delivery was not sent; with fallback the default envelope
            was sent instead.
        webhook_description (str | Unset): Description of that webhook. Set in delivery listings.
        webhook_url (str | Unset): URL of that webhook. Set in delivery listings.
    """

    attempt_count: int
    created_at: str
    delivery_id: str
    event_id: str
    max_attempts: int
    status: DeliveryItemStatus
    webhook_id: str
    schema: str | Unset = UNSET
    consumer: str | Unset = UNSET
    error_category: DeliveryItemErrorCategory | Unset = UNSET
    error_message: str | Unset = UNSET
    event_name: str | Unset = UNSET
    last_attempted_at: str | Unset = UNSET
    next_retry_at: str | Unset = UNSET
    response_body: str | Unset = UNSET
    response_code: int | Unset = UNSET
    template_error: str | Unset = UNSET
    webhook_description: str | Unset = UNSET
    webhook_url: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        attempt_count = self.attempt_count

        created_at = self.created_at

        delivery_id = self.delivery_id

        event_id = self.event_id

        max_attempts = self.max_attempts

        status = self.status.value

        webhook_id = self.webhook_id

        schema = self.schema

        consumer = self.consumer

        error_category: str | Unset = UNSET
        if not isinstance(self.error_category, Unset):
            error_category = self.error_category.value

        error_message = self.error_message

        event_name = self.event_name

        last_attempted_at = self.last_attempted_at

        next_retry_at = self.next_retry_at

        response_body = self.response_body

        response_code = self.response_code

        template_error = self.template_error

        webhook_description = self.webhook_description

        webhook_url = self.webhook_url

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "attempt_count": attempt_count,
                "created_at": created_at,
                "delivery_id": delivery_id,
                "event_id": event_id,
                "max_attempts": max_attempts,
                "status": status,
                "webhook_id": webhook_id,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if consumer is not UNSET:
            field_dict["consumer"] = consumer
        if error_category is not UNSET:
            field_dict["error_category"] = error_category
        if error_message is not UNSET:
            field_dict["error_message"] = error_message
        if event_name is not UNSET:
            field_dict["event_name"] = event_name
        if last_attempted_at is not UNSET:
            field_dict["last_attempted_at"] = last_attempted_at
        if next_retry_at is not UNSET:
            field_dict["next_retry_at"] = next_retry_at
        if response_body is not UNSET:
            field_dict["response_body"] = response_body
        if response_code is not UNSET:
            field_dict["response_code"] = response_code
        if template_error is not UNSET:
            field_dict["template_error"] = template_error
        if webhook_description is not UNSET:
            field_dict["webhook_description"] = webhook_description
        if webhook_url is not UNSET:
            field_dict["webhook_url"] = webhook_url

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        attempt_count = d.pop("attempt_count")

        created_at = d.pop("created_at")

        delivery_id = d.pop("delivery_id")

        event_id = d.pop("event_id")

        max_attempts = d.pop("max_attempts")

        status = DeliveryItemStatus(d.pop("status"))

        webhook_id = d.pop("webhook_id")

        schema = d.pop("$schema", UNSET)

        consumer = d.pop("consumer", UNSET)

        _error_category = d.pop("error_category", UNSET)
        error_category: DeliveryItemErrorCategory | Unset
        if isinstance(_error_category, Unset):
            error_category = UNSET
        else:
            error_category = DeliveryItemErrorCategory(_error_category)

        error_message = d.pop("error_message", UNSET)

        event_name = d.pop("event_name", UNSET)

        last_attempted_at = d.pop("last_attempted_at", UNSET)

        next_retry_at = d.pop("next_retry_at", UNSET)

        response_body = d.pop("response_body", UNSET)

        response_code = d.pop("response_code", UNSET)

        template_error = d.pop("template_error", UNSET)

        webhook_description = d.pop("webhook_description", UNSET)

        webhook_url = d.pop("webhook_url", UNSET)

        delivery_item = cls(
            attempt_count=attempt_count,
            created_at=created_at,
            delivery_id=delivery_id,
            event_id=event_id,
            max_attempts=max_attempts,
            status=status,
            webhook_id=webhook_id,
            schema=schema,
            consumer=consumer,
            error_category=error_category,
            error_message=error_message,
            event_name=event_name,
            last_attempted_at=last_attempted_at,
            next_retry_at=next_retry_at,
            response_body=response_body,
            response_code=response_code,
            template_error=template_error,
            webhook_description=webhook_description,
            webhook_url=webhook_url,
        )

        return delivery_item
