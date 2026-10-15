from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="CreateTokenBody")


@_attrs_define
class CreateTokenBody:
    """
    Attributes:
        name (str): Who or what the token is for, e.g. "alice" or "ci-deploy".
        schema (str | Unset): A URL to the JSON Schema for this object.
        consumer (str | Unset): Limit the token to one consumer: it then works only through the portal API
            (/portal/api/). Omit for a tenant-wide token with the same power as SPARROW_API_KEY. At most 255 characters, no
            /, \\, ?, #, % or dot segments; _sparrow is not allowed.
        external_id (str | Unset): Consumer tokens only. Your id for who or what the token is for, e.g. your user id
            when embedding the portal. There is at most one active token per consumer and external_id: while it is valid,
            asking again returns it, secret included (reused: true; name and ttl_seconds are ignored), instead of creating
            another; once it expires or is revoked, a new one is created. This makes minting a portal link on every page
            view safe. Not a secret: it is fine to log.
        never_expires (bool | Unset): Create a tenant-wide token that never expires (revoke it when no longer needed).
            Not allowed with ttl_seconds or for consumer tokens.
        ttl_seconds (int | Unset): Lifetime in seconds. 0 or omitted takes the default: SPARROW_TOKEN_DEFAULT_TTL (90
            days unless changed) for tenant-wide tokens, 7 days for consumer tokens. Consumer tokens allow at most 30 days.
    """

    name: str
    schema: str | Unset = UNSET
    consumer: str | Unset = UNSET
    external_id: str | Unset = UNSET
    never_expires: bool | Unset = UNSET
    ttl_seconds: int | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        schema = self.schema

        consumer = self.consumer

        external_id = self.external_id

        never_expires = self.never_expires

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
        if external_id is not UNSET:
            field_dict["external_id"] = external_id
        if never_expires is not UNSET:
            field_dict["never_expires"] = never_expires
        if ttl_seconds is not UNSET:
            field_dict["ttl_seconds"] = ttl_seconds

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        schema = d.pop("$schema", UNSET)

        consumer = d.pop("consumer", UNSET)

        external_id = d.pop("external_id", UNSET)

        never_expires = d.pop("never_expires", UNSET)

        ttl_seconds = d.pop("ttl_seconds", UNSET)

        create_token_body = cls(
            name=name,
            schema=schema,
            consumer=consumer,
            external_id=external_id,
            never_expires=never_expires,
            ttl_seconds=ttl_seconds,
        )

        return create_token_body
