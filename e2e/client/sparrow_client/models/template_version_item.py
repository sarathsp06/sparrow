from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..models.template_version_item_source import TemplateVersionItemSource
from ..types import UNSET, Unset

T = TypeVar("T", bound="TemplateVersionItem")


@_attrs_define
class TemplateVersionItem:
    """
    Attributes:
        created_at (datetime.datetime): When it was saved, RFC3339.
        current (bool): True for the newest version, which is the subscription's current template.
        source (TemplateVersionItemSource): How the template was produced: manual or ai_draft.
        template (str): The template text as saved.
        version_id (str): Version id (UUID).
        notes (str | Unset): Note stored with the version.
        saved_by (str | Unset): Name of the master key or access token that saved it; absent when auth is off.
    """

    created_at: datetime.datetime
    current: bool
    source: TemplateVersionItemSource
    template: str
    version_id: str
    notes: str | Unset = UNSET
    saved_by: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        current = self.current

        source = self.source.value

        template = self.template

        version_id = self.version_id

        notes = self.notes

        saved_by = self.saved_by

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "created_at": created_at,
                "current": current,
                "source": source,
                "template": template,
                "version_id": version_id,
            }
        )
        if notes is not UNSET:
            field_dict["notes"] = notes
        if saved_by is not UNSET:
            field_dict["saved_by"] = saved_by

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))

        current = d.pop("current")

        source = TemplateVersionItemSource(d.pop("source"))

        template = d.pop("template")

        version_id = d.pop("version_id")

        notes = d.pop("notes", UNSET)

        saved_by = d.pop("saved_by", UNSET)

        template_version_item = cls(
            created_at=created_at,
            current=current,
            source=source,
            template=template,
            version_id=version_id,
            notes=notes,
            saved_by=saved_by,
        )

        return template_version_item
