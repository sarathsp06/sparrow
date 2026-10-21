from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="CrlOut")


@_attrs_define
class CrlOut:
    """
    Attributes:
        entries (int): Revoked certificates it lists.
        issuer (str): Common name of the CA that signs this CRL.
        url (str): CRL distribution point, from an intermediate in a receiver's chain.
        error (str | Unset): Why the last download or check failed. In soft mode such a CRL gives no answer and
            connections are allowed.
        fetched_at (datetime.datetime | Unset): When this instance last downloaded it successfully.
        next_update (datetime.datetime | Unset): When the CA promises a newer CRL. Past it, the CRL is downloaded again
            before use.
    """

    entries: int
    issuer: str
    url: str
    error: str | Unset = UNSET
    fetched_at: datetime.datetime | Unset = UNSET
    next_update: datetime.datetime | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        entries = self.entries

        issuer = self.issuer

        url = self.url

        error = self.error

        fetched_at: str | Unset = UNSET
        if not isinstance(self.fetched_at, Unset):
            fetched_at = self.fetched_at.isoformat()

        next_update: str | Unset = UNSET
        if not isinstance(self.next_update, Unset):
            next_update = self.next_update.isoformat()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "entries": entries,
                "issuer": issuer,
                "url": url,
            }
        )
        if error is not UNSET:
            field_dict["error"] = error
        if fetched_at is not UNSET:
            field_dict["fetched_at"] = fetched_at
        if next_update is not UNSET:
            field_dict["next_update"] = next_update

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        entries = d.pop("entries")

        issuer = d.pop("issuer")

        url = d.pop("url")

        error = d.pop("error", UNSET)

        _fetched_at = d.pop("fetched_at", UNSET)
        fetched_at: datetime.datetime | Unset
        if isinstance(_fetched_at, Unset):
            fetched_at = UNSET
        else:
            fetched_at = datetime.datetime.fromisoformat(_fetched_at)

        _next_update = d.pop("next_update", UNSET)
        next_update: datetime.datetime | Unset
        if isinstance(_next_update, Unset):
            next_update = UNSET
        else:
            next_update = datetime.datetime.fromisoformat(_next_update)

        crl_out = cls(
            entries=entries,
            issuer=issuer,
            url=url,
            error=error,
            fetched_at=fetched_at,
            next_update=next_update,
        )

        return crl_out
