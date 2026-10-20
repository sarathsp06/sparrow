from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="GenerateReceiverCodeInputBody")


@_attrs_define
class GenerateReceiverCodeInputBody:
    """
    Attributes:
        framework (str): Target framework: nethttp (go), fastapi (python) or express (node).
        language (str): Target language: go, python or node (see GET /v1/capabilities, receiver_code.targets).
        schema (str | Unset): A URL to the JSON Schema for this object.
        subscription_ids (list[str] | Unset): Subscriptions to generate handlers for; omit for every subscription of the
            webhook.
    """

    framework: str
    language: str
    schema: str | Unset = UNSET
    subscription_ids: list[str] | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        framework = self.framework

        language = self.language

        schema = self.schema

        subscription_ids: list[str] | Unset = UNSET
        if not isinstance(self.subscription_ids, Unset):
            subscription_ids = self.subscription_ids

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "framework": framework,
                "language": language,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if subscription_ids is not UNSET:
            field_dict["subscription_ids"] = subscription_ids

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        framework = d.pop("framework")

        language = d.pop("language")

        schema = d.pop("$schema", UNSET)

        subscription_ids = cast(list[str], d.pop("subscription_ids", UNSET))

        generate_receiver_code_input_body = cls(
            framework=framework,
            language=language,
            schema=schema,
            subscription_ids=subscription_ids,
        )

        return generate_receiver_code_input_body
