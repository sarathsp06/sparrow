from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.token_out import TokenOut


T = TypeVar("T", bound="CreateTokenOutputBody")


@_attrs_define
class CreateTokenOutputBody:
    """
    Attributes:
        reused (bool): True when external_id matched a still-valid token, which is returned instead of a new one.
        secret (str): The credential. Shown only in this response (and again for the same external_id); send it as
            X-API-Key or Authorization: Bearer.
        token (TokenOut):
        schema (str | Unset): A URL to the JSON Schema for this object.
        portal_path (str | Unset): Consumer tokens only: server-relative portal URL with the token in the fragment
            (never sent to the server or logged). Prepend the base URL the UI is served from (the Sparrow server with
            SPARROW_SERVE_UI=true, or your separately hosted UI) and hand it to the end consumer.
    """

    reused: bool
    secret: str
    token: TokenOut
    schema: str | Unset = UNSET
    portal_path: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        reused = self.reused

        secret = self.secret

        token = self.token.to_dict()

        schema = self.schema

        portal_path = self.portal_path

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "reused": reused,
                "secret": secret,
                "token": token,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if portal_path is not UNSET:
            field_dict["portal_path"] = portal_path

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.token_out import TokenOut  # noqa: PLC0415

        d = dict(src_dict)
        reused = d.pop("reused")

        secret = d.pop("secret")

        token = TokenOut.from_dict(d.pop("token"))

        schema = d.pop("$schema", UNSET)

        portal_path = d.pop("portal_path", UNSET)

        create_token_output_body = cls(
            reused=reused,
            secret=secret,
            token=token,
            schema=schema,
            portal_path=portal_path,
        )

        return create_token_output_body
