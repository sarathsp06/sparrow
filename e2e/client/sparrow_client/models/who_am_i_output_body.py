from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="WhoAmIOutputBody")


@_attrs_define
class WhoAmIOutputBody:
    """
    Attributes:
        auth_enabled (bool): False when SPARROW_API_KEY is unset: every endpoint is open.
        master_key (bool): True when the request used SPARROW_API_KEY itself.
        name (str): Name of the credential used for this request.
        token_id (None | str): Id of the access token used, if any.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    auth_enabled: bool
    master_key: bool
    name: str
    token_id: None | str
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        auth_enabled = self.auth_enabled

        master_key = self.master_key

        name = self.name

        token_id: None | str
        token_id = self.token_id

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "auth_enabled": auth_enabled,
                "master_key": master_key,
                "name": name,
                "token_id": token_id,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        auth_enabled = d.pop("auth_enabled")

        master_key = d.pop("master_key")

        name = d.pop("name")

        def _parse_token_id(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        token_id = _parse_token_id(d.pop("token_id"))

        schema = d.pop("$schema", UNSET)

        who_am_i_output_body = cls(
            auth_enabled=auth_enabled,
            master_key=master_key,
            name=name,
            token_id=token_id,
            schema=schema,
        )

        return who_am_i_output_body
