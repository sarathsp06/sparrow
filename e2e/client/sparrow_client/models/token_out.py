from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..models.token_out_status import TokenOutStatus

T = TypeVar("T", bound="TokenOut")


@_attrs_define
class TokenOut:
    """
    Attributes:
        consumer (None | str): Consumer the token is limited to (portal API only). Null: tenant-wide, same power as
            SPARROW_API_KEY.
        created_at (datetime.datetime): When the token was created, RFC3339.
        created_by (str): Name of the credential that created it (the master key, or another token).
        expires_at (datetime.datetime | None): When the token stops working, RFC3339. Null: never expires.
        id (str): Token id (not the secret).
        last_used_at (datetime.datetime | None): When the token was last used, RFC3339; approximate (updated at most
            once a minute). Null: never used.
        name (str): Who or what the token is for.
        revoked_at (datetime.datetime | None): When the token was revoked, RFC3339. Null: not revoked.
        status (TokenOutStatus): active: usable. revoked: revoked by DELETE /v1/tokens/{token_id}. expired: past
            expires_at.
    """

    consumer: None | str
    created_at: datetime.datetime
    created_by: str
    expires_at: datetime.datetime | None
    id: str
    last_used_at: datetime.datetime | None
    name: str
    revoked_at: datetime.datetime | None
    status: TokenOutStatus

    def to_dict(self) -> dict[str, Any]:
        consumer: None | str
        consumer = self.consumer

        created_at = self.created_at.isoformat()

        created_by = self.created_by

        expires_at: None | str
        if isinstance(self.expires_at, datetime.datetime):
            expires_at = self.expires_at.isoformat()
        else:
            expires_at = self.expires_at

        id = self.id

        last_used_at: None | str
        if isinstance(self.last_used_at, datetime.datetime):
            last_used_at = self.last_used_at.isoformat()
        else:
            last_used_at = self.last_used_at

        name = self.name

        revoked_at: None | str
        if isinstance(self.revoked_at, datetime.datetime):
            revoked_at = self.revoked_at.isoformat()
        else:
            revoked_at = self.revoked_at

        status = self.status.value

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "consumer": consumer,
                "created_at": created_at,
                "created_by": created_by,
                "expires_at": expires_at,
                "id": id,
                "last_used_at": last_used_at,
                "name": name,
                "revoked_at": revoked_at,
                "status": status,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)

        def _parse_consumer(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        consumer = _parse_consumer(d.pop("consumer"))

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))

        created_by = d.pop("created_by")

        def _parse_expires_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                expires_at_type_0 = datetime.datetime.fromisoformat(data)

                return expires_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        expires_at = _parse_expires_at(d.pop("expires_at"))

        id = d.pop("id")

        def _parse_last_used_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_used_at_type_0 = datetime.datetime.fromisoformat(data)

                return last_used_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        last_used_at = _parse_last_used_at(d.pop("last_used_at"))

        name = d.pop("name")

        def _parse_revoked_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                revoked_at_type_0 = datetime.datetime.fromisoformat(data)

                return revoked_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        revoked_at = _parse_revoked_at(d.pop("revoked_at"))

        status = TokenOutStatus(d.pop("status"))

        token_out = cls(
            consumer=consumer,
            created_at=created_at,
            created_by=created_by,
            expires_at=expires_at,
            id=id,
            last_used_at=last_used_at,
            name=name,
            revoked_at=revoked_at,
            status=status,
        )

        return token_out
