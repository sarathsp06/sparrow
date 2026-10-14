from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

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
        description (str):
        name (str):
        subscription (Subscription):
        version (int):
        webhook (Webhook):
        consumer (ConsumerHint | Unset):
        guidance (str | Unset): What an ideal message for this destination looks like. Advice for template authors and
            AI drafting, not a rule.
        params (list[Param] | None | Unset):
    """

    description: str
    name: str
    subscription: Subscription
    version: int
    webhook: Webhook
    consumer: ConsumerHint | Unset = UNSET
    guidance: str | Unset = UNSET
    params: list[Param] | None | Unset = UNSET

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

        params: list[dict[str, Any]] | None | Unset
        if isinstance(self.params, Unset):
            params = UNSET
        elif isinstance(self.params, list):
            params = []
            for params_type_0_item_data in self.params:
                params_type_0_item = params_type_0_item_data.to_dict()
                params.append(params_type_0_item)

        else:
            params = self.params

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

        def _parse_params(data: object) -> list[Param] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                params_type_0 = []
                _params_type_0 = data
                for params_type_0_item_data in _params_type_0:
                    params_type_0_item = Param.from_dict(params_type_0_item_data)

                    params_type_0.append(params_type_0_item)

                return params_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[Param] | None | Unset, data)

        params = _parse_params(d.pop("params", UNSET))

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
