from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..models.template_failure_item_payload import TemplateFailureItemPayload

T = TypeVar("T", bound="TemplateFailureItem")


@_attrs_define
class TemplateFailureItem:
    """
    Attributes:
        consumer (str): Consumer the subscription belongs to.
        error (str): The render error.
        payload (TemplateFailureItemPayload): Which generated payload failed: every field, or only required fields (a
            template reading an optional field without guarding for it).
        subscription_id (str): Subscription whose template failed.
        webhook_id (str): Webhook the subscription delivers to.
    """

    consumer: str
    error: str
    payload: TemplateFailureItemPayload
    subscription_id: str
    webhook_id: str

    def to_dict(self) -> dict[str, Any]:
        consumer = self.consumer

        error = self.error

        payload = self.payload.value

        subscription_id = self.subscription_id

        webhook_id = self.webhook_id

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "consumer": consumer,
                "error": error,
                "payload": payload,
                "subscription_id": subscription_id,
                "webhook_id": webhook_id,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        consumer = d.pop("consumer")

        error = d.pop("error")

        payload = TemplateFailureItemPayload(d.pop("payload"))

        subscription_id = d.pop("subscription_id")

        webhook_id = d.pop("webhook_id")

        template_failure_item = cls(
            consumer=consumer,
            error=error,
            payload=payload,
            subscription_id=subscription_id,
            webhook_id=webhook_id,
        )

        return template_failure_item
