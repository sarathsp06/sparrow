from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateInviteBody")


@_attrs_define
class CreateInviteBody:
    """
    Attributes:
        name (str): Name for the token the invite creates, e.g. the invitee's name.
        schema (str | Unset): A URL to the JSON Schema for this object.
        consumer (str | Unset): Invite into one consumer's portal. Omit for tenant-wide (operator console) access.
        token_never_expires (bool | Unset): The token it creates never expires; same rules as never_expires on POST
            /v1/tokens.
        token_ttl_seconds (int | Unset): Lifetime of the token it creates; same defaults and limits as POST /v1/tokens.
        ttl_seconds (int | Unset): How long the invite can be used. Defaults to 24 hours, at most 7 days.
    """

    name: str
    schema: str | Unset = UNSET
    consumer: str | Unset = UNSET
    token_never_expires: bool | Unset = UNSET
    token_ttl_seconds: int | Unset = UNSET
    ttl_seconds: int | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        schema = self.schema

        consumer = self.consumer

        token_never_expires = self.token_never_expires

        token_ttl_seconds = self.token_ttl_seconds

        ttl_seconds = self.ttl_seconds

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "name": name,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if consumer is not UNSET:
            field_dict["consumer"] = consumer
        if token_never_expires is not UNSET:
            field_dict["token_never_expires"] = token_never_expires
        if token_ttl_seconds is not UNSET:
            field_dict["token_ttl_seconds"] = token_ttl_seconds
        if ttl_seconds is not UNSET:
            field_dict["ttl_seconds"] = ttl_seconds

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        schema = d.pop("$schema", UNSET)

        consumer = d.pop("consumer", UNSET)

        token_never_expires = d.pop("token_never_expires", UNSET)

        token_ttl_seconds = d.pop("token_ttl_seconds", UNSET)

        ttl_seconds = d.pop("ttl_seconds", UNSET)

        create_invite_body = cls(
            name=name,
            schema=schema,
            consumer=consumer,
            token_never_expires=token_never_expires,
            token_ttl_seconds=token_ttl_seconds,
            ttl_seconds=ttl_seconds,
        )

        return create_invite_body
