from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="ExportEventTypesInputBody")


@_attrs_define
class ExportEventTypesInputBody:
    """
    Attributes:
        schema (str | Unset): A URL to the JSON Schema for this object.
        all_ (bool | Unset): Export every event type.
        names (list[str] | None | Unset): Export these event types. Each must exist.
        prefix (str | Unset): Export every event type whose name starts with this prefix.
    """

    schema: str | Unset = UNSET
    all_: bool | Unset = UNSET
    names: list[str] | None | Unset = UNSET
    prefix: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        schema = self.schema

        all_ = self.all_

        names: list[str] | None | Unset
        if isinstance(self.names, Unset):
            names = UNSET
        elif isinstance(self.names, list):
            names = self.names

        else:
            names = self.names

        prefix = self.prefix

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if all_ is not UNSET:
            field_dict["all"] = all_
        if names is not UNSET:
            field_dict["names"] = names
        if prefix is not UNSET:
            field_dict["prefix"] = prefix

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        schema = d.pop("$schema", UNSET)

        all_ = d.pop("all", UNSET)

        def _parse_names(data: object) -> list[str] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                names_type_0 = cast(list[str], data)

                return names_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None | Unset, data)

        names = _parse_names(d.pop("names", UNSET))

        prefix = d.pop("prefix", UNSET)

        export_event_types_input_body = cls(
            schema=schema,
            all_=all_,
            names=names,
            prefix=prefix,
        )

        return export_event_types_input_body
