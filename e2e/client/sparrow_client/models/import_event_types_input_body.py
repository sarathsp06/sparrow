from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..models.import_event_types_input_body_acknowledge_item import ImportEventTypesInputBodyAcknowledgeItem
from ..models.import_event_types_input_body_api_version import ImportEventTypesInputBodyApiVersion
from ..models.import_event_types_input_body_kind import ImportEventTypesInputBodyKind
from ..models.import_event_types_input_body_subscription_policy import ImportEventTypesInputBodySubscriptionPolicy
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.bundle_item import BundleItem
    from ..models.bundle_stamp_body import BundleStampBody


T = TypeVar("T", bound="ImportEventTypesInputBody")


@_attrs_define
class ImportEventTypesInputBody:
    """
    Attributes:
        items (list[BundleItem]): Definitions to import. Each replaces the named event type; types not listed are never
            touched.
        schema (str | Unset): A URL to the JSON Schema for this object.
        acknowledge (list[ImportEventTypesInputBodyAcknowledgeItem] | Unset): Stamp warnings you accept. Each
            unacknowledged warning blocks the write.
        allow_breaking (bool | Unset): Apply breaking schema changes to event types that subscriptions receive.
        api_version (ImportEventTypesInputBodyApiVersion | Unset): Bundle schema version, if the body is an exported
            bundle.
        dry_run (bool | Unset): Compute the full result without writing anything.
        kind (ImportEventTypesInputBodyKind | Unset): Bundle kind, if the body is an exported bundle.
        stamp (BundleStampBody | Unset):
        subscription_policy (ImportEventTypesInputBodySubscriptionPolicy | Unset): keep_active (default): subscriptions
            keep running; a template that no longer fits fails its deliveries visibly with template_error. pause: in the
            same transaction, pause each subscription whose template fails the pre-check, with the import as the reason.
    """

    items: list[BundleItem]
    schema: str | Unset = UNSET
    acknowledge: list[ImportEventTypesInputBodyAcknowledgeItem] | Unset = UNSET
    allow_breaking: bool | Unset = UNSET
    api_version: ImportEventTypesInputBodyApiVersion | Unset = UNSET
    dry_run: bool | Unset = UNSET
    kind: ImportEventTypesInputBodyKind | Unset = UNSET
    stamp: BundleStampBody | Unset = UNSET
    subscription_policy: ImportEventTypesInputBodySubscriptionPolicy | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        schema = self.schema

        acknowledge: list[str] | Unset = UNSET
        if not isinstance(self.acknowledge, Unset):
            acknowledge = []
            for acknowledge_item_data in self.acknowledge:
                acknowledge_item = acknowledge_item_data.value
                acknowledge.append(acknowledge_item)

        allow_breaking = self.allow_breaking

        api_version: str | Unset = UNSET
        if not isinstance(self.api_version, Unset):
            api_version = self.api_version.value

        dry_run = self.dry_run

        kind: str | Unset = UNSET
        if not isinstance(self.kind, Unset):
            kind = self.kind.value

        stamp: dict[str, Any] | Unset = UNSET
        if not isinstance(self.stamp, Unset):
            stamp = self.stamp.to_dict()

        subscription_policy: str | Unset = UNSET
        if not isinstance(self.subscription_policy, Unset):
            subscription_policy = self.subscription_policy.value

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "items": items,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if acknowledge is not UNSET:
            field_dict["acknowledge"] = acknowledge
        if allow_breaking is not UNSET:
            field_dict["allow_breaking"] = allow_breaking
        if api_version is not UNSET:
            field_dict["apiVersion"] = api_version
        if dry_run is not UNSET:
            field_dict["dry_run"] = dry_run
        if kind is not UNSET:
            field_dict["kind"] = kind
        if stamp is not UNSET:
            field_dict["stamp"] = stamp
        if subscription_policy is not UNSET:
            field_dict["subscription_policy"] = subscription_policy

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.bundle_item import BundleItem  # noqa: PLC0415
        from ..models.bundle_stamp_body import BundleStampBody  # noqa: PLC0415

        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = BundleItem.from_dict(items_item_data)

            items.append(items_item)

        schema = d.pop("$schema", UNSET)

        _acknowledge = d.pop("acknowledge", UNSET)
        acknowledge: list[ImportEventTypesInputBodyAcknowledgeItem] | Unset = UNSET
        if _acknowledge is not UNSET:
            acknowledge = []
            for acknowledge_item_data in _acknowledge:
                acknowledge_item = ImportEventTypesInputBodyAcknowledgeItem(acknowledge_item_data)

                acknowledge.append(acknowledge_item)

        allow_breaking = d.pop("allow_breaking", UNSET)

        _api_version = d.pop("apiVersion", UNSET)
        api_version: ImportEventTypesInputBodyApiVersion | Unset
        if isinstance(_api_version, Unset):
            api_version = UNSET
        else:
            api_version = ImportEventTypesInputBodyApiVersion(_api_version)

        dry_run = d.pop("dry_run", UNSET)

        _kind = d.pop("kind", UNSET)
        kind: ImportEventTypesInputBodyKind | Unset
        if isinstance(_kind, Unset):
            kind = UNSET
        else:
            kind = ImportEventTypesInputBodyKind(_kind)

        _stamp = d.pop("stamp", UNSET)
        stamp: BundleStampBody | Unset
        if isinstance(_stamp, Unset):
            stamp = UNSET
        else:
            stamp = BundleStampBody.from_dict(_stamp)

        _subscription_policy = d.pop("subscription_policy", UNSET)
        subscription_policy: ImportEventTypesInputBodySubscriptionPolicy | Unset
        if isinstance(_subscription_policy, Unset):
            subscription_policy = UNSET
        else:
            subscription_policy = ImportEventTypesInputBodySubscriptionPolicy(_subscription_policy)

        import_event_types_input_body = cls(
            items=items,
            schema=schema,
            acknowledge=acknowledge,
            allow_breaking=allow_breaking,
            api_version=api_version,
            dry_run=dry_run,
            kind=kind,
            stamp=stamp,
            subscription_policy=subscription_policy,
        )

        return import_event_types_input_body
