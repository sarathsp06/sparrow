from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..models.invite_out_status import InviteOutStatus

T = TypeVar("T", bound="InviteOut")


@_attrs_define
class InviteOut:
    """
    Attributes:
        cancelled_at (datetime.datetime | None): When the invite was cancelled, RFC3339. Null: not cancelled.
        consumer (None | str): Consumer the resulting token is limited to. Null: tenant-wide.
        created_at (datetime.datetime): When the invite was created, RFC3339.
        created_by (str): Name of the credential that created the invite.
        expires_at (datetime.datetime): When the invite stops working if unused, RFC3339.
        id (str): Invite id (not the secret).
        name (str): Name given to the token the invite creates.
        redeemed_at (datetime.datetime | None): When the invite was redeemed, RFC3339. Null: not redeemed.
        status (InviteOutStatus): pending: can still be redeemed. redeemed: used once, see token_id. cancelled:
            cancelled by DELETE /v1/invites/{invite_id}. expired: past expires_at unused.
        token_id (None | str): Token created when the invite was redeemed.
        token_ttl_seconds (int | None): Lifetime of the token it creates, in seconds. Null: the token never expires.
    """

    cancelled_at: datetime.datetime | None
    consumer: None | str
    created_at: datetime.datetime
    created_by: str
    expires_at: datetime.datetime
    id: str
    name: str
    redeemed_at: datetime.datetime | None
    status: InviteOutStatus
    token_id: None | str
    token_ttl_seconds: int | None

    def to_dict(self) -> dict[str, Any]:
        cancelled_at: None | str
        if isinstance(self.cancelled_at, datetime.datetime):
            cancelled_at = self.cancelled_at.isoformat()
        else:
            cancelled_at = self.cancelled_at

        consumer: None | str
        consumer = self.consumer

        created_at = self.created_at.isoformat()

        created_by = self.created_by

        expires_at = self.expires_at.isoformat()

        id = self.id

        name = self.name

        redeemed_at: None | str
        if isinstance(self.redeemed_at, datetime.datetime):
            redeemed_at = self.redeemed_at.isoformat()
        else:
            redeemed_at = self.redeemed_at

        status = self.status.value

        token_id: None | str
        token_id = self.token_id

        token_ttl_seconds: int | None
        token_ttl_seconds = self.token_ttl_seconds

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "cancelled_at": cancelled_at,
                "consumer": consumer,
                "created_at": created_at,
                "created_by": created_by,
                "expires_at": expires_at,
                "id": id,
                "name": name,
                "redeemed_at": redeemed_at,
                "status": status,
                "token_id": token_id,
                "token_ttl_seconds": token_ttl_seconds,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)

        def _parse_cancelled_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                cancelled_at_type_0 = datetime.datetime.fromisoformat(data)

                return cancelled_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        cancelled_at = _parse_cancelled_at(d.pop("cancelled_at"))

        def _parse_consumer(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        consumer = _parse_consumer(d.pop("consumer"))

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))

        created_by = d.pop("created_by")

        expires_at = datetime.datetime.fromisoformat(d.pop("expires_at"))

        id = d.pop("id")

        name = d.pop("name")

        def _parse_redeemed_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                redeemed_at_type_0 = datetime.datetime.fromisoformat(data)

                return redeemed_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        redeemed_at = _parse_redeemed_at(d.pop("redeemed_at"))

        status = InviteOutStatus(d.pop("status"))

        def _parse_token_id(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        token_id = _parse_token_id(d.pop("token_id"))

        def _parse_token_ttl_seconds(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        token_ttl_seconds = _parse_token_ttl_seconds(d.pop("token_ttl_seconds"))

        invite_out = cls(
            cancelled_at=cancelled_at,
            consumer=consumer,
            created_at=created_at,
            created_by=created_by,
            expires_at=expires_at,
            id=id,
            name=name,
            redeemed_at=redeemed_at,
            status=status,
            token_id=token_id,
            token_ttl_seconds=token_ttl_seconds,
        )

        return invite_out
