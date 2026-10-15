from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.invite_out import InviteOut


T = TypeVar("T", bound="CreateInviteOutputBody")


@_attrs_define
class CreateInviteOutputBody:
    """
    Attributes:
        invite (InviteOut):
        path (str): UI-relative link with the secret in the fragment (never sent to servers or logs): /#invite=... for a
            tenant-wide invite, /portal#invite=... for a consumer invite. Prepend the base URL the UI is served from.
        secret (str): The invite secret. Shown only in this response.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    invite: InviteOut
    path: str
    secret: str
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        invite = self.invite.to_dict()

        path = self.path

        secret = self.secret

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "invite": invite,
                "path": path,
                "secret": secret,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.invite_out import InviteOut  # noqa: PLC0415

        d = dict(src_dict)
        invite = InviteOut.from_dict(d.pop("invite"))

        path = d.pop("path")

        secret = d.pop("secret")

        schema = d.pop("$schema", UNSET)

        create_invite_output_body = cls(
            invite=invite,
            path=path,
            secret=secret,
            schema=schema,
        )

        return create_invite_output_body
