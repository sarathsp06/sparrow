from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.consumer_hint import ConsumerHint
    from ..models.param import Param
    from ..models.subscription import Subscription
    from ..models.webhook import Webhook


T = TypeVar("T", bound="Recipe")


@_attrs_define
class Recipe:
    """
    Attributes:
        description (str): One-line summary of the destination the recipe targets.
        name (str): Recipe name, e.g. slack. Unique among shipped recipes.
        subscription (Subscription):
        version (int): Recipe schema version. Always 1.
        webhook (Webhook):
        consumer (ConsumerHint | Unset):
        guidance (str | Unset): What an ideal message for this destination looks like. Advice for template authors and
            AI drafting, not a rule.
        params (list[Param] | Unset): Values the user supplies when applying the recipe. Each {{param "name"}} token in
            webhook.url, webhook.headers, webhook.secret_headers and subscription.transform_template is replaced with the
            value by plain string substitution before registration.
    """

    description: str
    name: str
    subscription: Subscription
    version: int
    webhook: Webhook
    consumer: ConsumerHint | Unset = UNSET
    guidance: str | Unset = UNSET
    params: list[Param] | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        description = self.description

        name = self.name

        subscription = self.subscription.to_dict()

        version = self.version

        webhook = self.webhook.to_dict()

        consumer: dict[str, Any] | Unset = UNSET
        if not isinstance(self.consumer, Unset):
            consumer = self.consumer.to_dict()

        guidance = self.guidance

        params: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.params, Unset):
            params = []
            for params_item_data in self.params:
                params_item = params_item_data.to_dict()
                params.append(params_item)

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "description": description,
                "name": name,
                "subscription": subscription,
                "version": version,
                "webhook": webhook,
            }
        )
        if consumer is not UNSET:
            field_dict["consumer"] = consumer
        if guidance is not UNSET:
            field_dict["guidance"] = guidance
        if params is not UNSET:
            field_dict["params"] = params

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.consumer_hint import ConsumerHint  # noqa: PLC0415
        from ..models.param import Param  # noqa: PLC0415
        from ..models.subscription import Subscription  # noqa: PLC0415
        from ..models.webhook import Webhook  # noqa: PLC0415

        d = dict(src_dict)
        description = d.pop("description")

        name = d.pop("name")

        subscription = Subscription.from_dict(d.pop("subscription"))

        version = d.pop("version")

        webhook = Webhook.from_dict(d.pop("webhook"))

        _consumer = d.pop("consumer", UNSET)
        consumer: ConsumerHint | Unset
        if isinstance(_consumer, Unset):
            consumer = UNSET
        else:
            consumer = ConsumerHint.from_dict(_consumer)

        guidance = d.pop("guidance", UNSET)

        _params = d.pop("params", UNSET)
        params: list[Param] | Unset = UNSET
        if _params is not UNSET:
            params = []
            for params_item_data in _params:
                params_item = Param.from_dict(params_item_data)

                params.append(params_item)

        recipe = cls(
            description=description,
            name=name,
            subscription=subscription,
            version=version,
            webhook=webhook,
            consumer=consumer,
            guidance=guidance,
            params=params,
        )

        return recipe
