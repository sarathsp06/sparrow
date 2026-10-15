from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define

from ..models.import_item_result_action import ImportItemResultAction
from ..models.import_item_result_active_change import ImportItemResultActiveChange
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.schema_compatibility_item import SchemaCompatibilityItem
    from ..models.template_check_item import TemplateCheckItem


T = TypeVar("T", bound="ImportItemResult")


@_attrs_define
class ImportItemResult:
    """
    Attributes:
        action (ImportItemResultAction): What the import does to this event type. See updateEventType's change.action.
        name (str): Event type name.
        version (int): Version after the import (or that it would have, for a dry run).
        active_change (ImportItemResultActiveChange | Unset): Set when the import flips the active flag.
        blocked (bool | Unset): The change is breaking for subscriptions and allow_breaking was not set.
        changes (list[str] | Unset): Fields that change: schema, schema_defined, description, metadata, active.
        compatibility (SchemaCompatibilityItem | Unset):
        previous_version (int | Unset): Version before the import.
        subscriptions (TemplateCheckItem | Unset):
    """

    action: ImportItemResultAction
    name: str
    version: int
    active_change: ImportItemResultActiveChange | Unset = UNSET
    blocked: bool | Unset = UNSET
    changes: list[str] | Unset = UNSET
    compatibility: SchemaCompatibilityItem | Unset = UNSET
    previous_version: int | Unset = UNSET
    subscriptions: TemplateCheckItem | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        action = self.action.value

        name = self.name

        version = self.version

        active_change: str | Unset = UNSET
        if not isinstance(self.active_change, Unset):
            active_change = self.active_change.value

        blocked = self.blocked

        changes: list[str] | Unset = UNSET
        if not isinstance(self.changes, Unset):
            changes = self.changes

        compatibility: dict[str, Any] | Unset = UNSET
        if not isinstance(self.compatibility, Unset):
            compatibility = self.compatibility.to_dict()

        previous_version = self.previous_version

        subscriptions: dict[str, Any] | Unset = UNSET
        if not isinstance(self.subscriptions, Unset):
            subscriptions = self.subscriptions.to_dict()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "action": action,
                "name": name,
                "version": version,
            }
        )
        if active_change is not UNSET:
            field_dict["active_change"] = active_change
        if blocked is not UNSET:
            field_dict["blocked"] = blocked
        if changes is not UNSET:
            field_dict["changes"] = changes
        if compatibility is not UNSET:
            field_dict["compatibility"] = compatibility
        if previous_version is not UNSET:
            field_dict["previous_version"] = previous_version
        if subscriptions is not UNSET:
            field_dict["subscriptions"] = subscriptions

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.schema_compatibility_item import SchemaCompatibilityItem  # noqa: PLC0415
        from ..models.template_check_item import TemplateCheckItem  # noqa: PLC0415

        d = dict(src_dict)
        action = ImportItemResultAction(d.pop("action"))

        name = d.pop("name")

        version = d.pop("version")

        _active_change = d.pop("active_change", UNSET)
        active_change: ImportItemResultActiveChange | Unset
        if isinstance(_active_change, Unset):
            active_change = UNSET
        else:
            active_change = ImportItemResultActiveChange(_active_change)

        blocked = d.pop("blocked", UNSET)

        changes = cast(list[str], d.pop("changes", UNSET))

        _compatibility = d.pop("compatibility", UNSET)
        compatibility: SchemaCompatibilityItem | Unset
        if isinstance(_compatibility, Unset):
            compatibility = UNSET
        else:
            compatibility = SchemaCompatibilityItem.from_dict(_compatibility)

        previous_version = d.pop("previous_version", UNSET)

        _subscriptions = d.pop("subscriptions", UNSET)
        subscriptions: TemplateCheckItem | Unset
        if isinstance(_subscriptions, Unset):
            subscriptions = UNSET
        else:
            subscriptions = TemplateCheckItem.from_dict(_subscriptions)

        import_item_result = cls(
            action=action,
            name=name,
            version=version,
            active_change=active_change,
            blocked=blocked,
            changes=changes,
            compatibility=compatibility,
            previous_version=previous_version,
            subscriptions=subscriptions,
        )

        return import_item_result
