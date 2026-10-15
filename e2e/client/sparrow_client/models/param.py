from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="Param")


@_attrs_define
class Param:
    """
    Attributes:
        name (str):
        prompt (str):
        required (bool):
        activation_required (bool | Unset):
        default (str | Unset):
        docs_url (str | Unset): Destination documentation for this value.
        enum (list[str] | None | Unset): The only accepted values, when set.
        example (str | Unset): A sample value.
        help_ (str | Unset): What the value is and where to find it.
        must_override_default (bool | Unset):
        secret (bool | Unset):
    """

    name: str
    prompt: str
    required: bool
    activation_required: bool | Unset = UNSET
    default: str | Unset = UNSET
    docs_url: str | Unset = UNSET
    enum: list[str] | None | Unset = UNSET
    example: str | Unset = UNSET
    help_: str | Unset = UNSET
    must_override_default: bool | Unset = UNSET
    secret: bool | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        prompt = self.prompt

        required = self.required

        activation_required = self.activation_required

        default = self.default

        docs_url = self.docs_url

        enum: list[str] | None | Unset
        if isinstance(self.enum, Unset):
            enum = UNSET
        elif isinstance(self.enum, list):
            enum = self.enum

        else:
            enum = self.enum

        example = self.example

        help_ = self.help_

        must_override_default = self.must_override_default

        secret = self.secret

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "name": name,
                "prompt": prompt,
                "required": required,
            }
        )
        if activation_required is not UNSET:
            field_dict["activation_required"] = activation_required
        if default is not UNSET:
            field_dict["default"] = default
        if docs_url is not UNSET:
            field_dict["docs_url"] = docs_url
        if enum is not UNSET:
            field_dict["enum"] = enum
        if example is not UNSET:
            field_dict["example"] = example
        if help_ is not UNSET:
            field_dict["help"] = help_
        if must_override_default is not UNSET:
            field_dict["must_override_default"] = must_override_default
        if secret is not UNSET:
            field_dict["secret"] = secret

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        prompt = d.pop("prompt")

        required = d.pop("required")

        activation_required = d.pop("activation_required", UNSET)

        default = d.pop("default", UNSET)

        docs_url = d.pop("docs_url", UNSET)

        def _parse_enum(data: object) -> list[str] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                enum_type_0 = cast(list[str], data)

                return enum_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None | Unset, data)

        enum = _parse_enum(d.pop("enum", UNSET))

        example = d.pop("example", UNSET)

        help_ = d.pop("help", UNSET)

        must_override_default = d.pop("must_override_default", UNSET)

        secret = d.pop("secret", UNSET)

        param = cls(
            name=name,
            prompt=prompt,
            required=required,
            activation_required=activation_required,
            default=default,
            docs_url=docs_url,
            enum=enum,
            example=example,
            help_=help_,
            must_override_default=must_override_default,
            secret=secret,
        )

        return param
