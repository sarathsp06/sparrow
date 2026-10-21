from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..models.list_crls_output_body_mode import ListCrlsOutputBodyMode
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.crl_out import CrlOut


T = TypeVar("T", bound="ListCrlsOutputBody")


@_attrs_define
class ListCrlsOutputBody:
    """
    Attributes:
        crls (list[CrlOut]): CRLs this instance has cached, by URL. Each instance keeps its own cache.
        mode (ListCrlsOutputBodyMode): SPARROW_TLS_REVOCATION: soft allows a connection whose CRL is unavailable, hard
            refuses it, off skips the check.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    crls: list[CrlOut]
    mode: ListCrlsOutputBodyMode
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        crls = []
        for crls_item_data in self.crls:
            crls_item = crls_item_data.to_dict()
            crls.append(crls_item)

        mode = self.mode.value

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "crls": crls,
                "mode": mode,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.crl_out import CrlOut  # noqa: PLC0415

        d = dict(src_dict)
        crls = []
        _crls = d.pop("crls")
        for crls_item_data in _crls:
            crls_item = CrlOut.from_dict(crls_item_data)

            crls.append(crls_item)

        mode = ListCrlsOutputBodyMode(d.pop("mode"))

        schema = d.pop("$schema", UNSET)

        list_crls_output_body = cls(
            crls=crls,
            mode=mode,
            schema=schema,
        )

        return list_crls_output_body
