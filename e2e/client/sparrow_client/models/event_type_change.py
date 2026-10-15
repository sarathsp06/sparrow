from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define

from ..models.event_type_change_action import EventTypeChangeAction
from ..models.event_type_change_active_change import EventTypeChangeActiveChange
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.schema_compatibility_item import SchemaCompatibilityItem


T = TypeVar("T", bound="EventTypeChange")


@_attrs_define
class EventTypeChange:
    """
    Attributes:
        action (EventTypeChangeAction): created: new event type at version 1. new_version: the schema changed, so a new
            version was created and the previous one kept. updated: the current version changed in place (description,
            metadata, active, or a first schema added to a schema-less version). unchanged: nothing was written.
        version (int): Version after the write.
        active_change (EventTypeChangeActiveChange | Unset): Set when the write flips the active flag.
        affected_subscriptions (int | Unset): For new_version: subscriptions that receive this event type, by name or
            catch-all.
        changes (list[str] | Unset): Fields that changed: schema, schema_defined (a first schema was added in place),
            description, metadata, active.
        compatibility (SchemaCompatibilityItem | Unset):
        previous_version (int | Unset): Version before the write; absent when the event type was created.
    """

    action: EventTypeChangeAction
    version: int
    active_change: EventTypeChangeActiveChange | Unset = UNSET
    affected_subscriptions: int | Unset = UNSET
    changes: list[str] | Unset = UNSET
    compatibility: SchemaCompatibilityItem | Unset = UNSET
    previous_version: int | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        action = self.action.value

        version = self.version

        active_change: str | Unset = UNSET
        if not isinstance(self.active_change, Unset):
            active_change = self.active_change.value

        affected_subscriptions = self.affected_subscriptions

        changes: list[str] | Unset = UNSET
        if not isinstance(self.changes, Unset):
            changes = self.changes

        compatibility: dict[str, Any] | Unset = UNSET
        if not isinstance(self.compatibility, Unset):
            compatibility = self.compatibility.to_dict()

        previous_version = self.previous_version

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "action": action,
                "version": version,
            }
        )
        if active_change is not UNSET:
            field_dict["active_change"] = active_change
        if affected_subscriptions is not UNSET:
            field_dict["affected_subscriptions"] = affected_subscriptions
        if changes is not UNSET:
            field_dict["changes"] = changes
        if compatibility is not UNSET:
            field_dict["compatibility"] = compatibility
        if previous_version is not UNSET:
            field_dict["previous_version"] = previous_version

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.schema_compatibility_item import SchemaCompatibilityItem  # noqa: PLC0415

        d = dict(src_dict)
        action = EventTypeChangeAction(d.pop("action"))

        version = d.pop("version")

        _active_change = d.pop("active_change", UNSET)
        active_change: EventTypeChangeActiveChange | Unset
        if isinstance(_active_change, Unset):
            active_change = UNSET
        else:
            active_change = EventTypeChangeActiveChange(_active_change)

        affected_subscriptions = d.pop("affected_subscriptions", UNSET)

        changes = cast(list[str], d.pop("changes", UNSET))

        _compatibility = d.pop("compatibility", UNSET)
        compatibility: SchemaCompatibilityItem | Unset
        if isinstance(_compatibility, Unset):
            compatibility = UNSET
        else:
            compatibility = SchemaCompatibilityItem.from_dict(_compatibility)

        previous_version = d.pop("previous_version", UNSET)

        event_type_change = cls(
            action=action,
            version=version,
            active_change=active_change,
            affected_subscriptions=affected_subscriptions,
            changes=changes,
            compatibility=compatibility,
            previous_version=previous_version,
        )

        return event_type_change
