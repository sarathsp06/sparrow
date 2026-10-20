from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

if TYPE_CHECKING:
    from ..models.receiver_target_out import ReceiverTargetOut


T = TypeVar("T", bound="ReceiverCodeStruct")


@_attrs_define
class ReceiverCodeStruct:
    """
    Attributes:
        targets (list[ReceiverTargetOut]): Language and framework pairs POST
            /v1/consumers/{consumer}/webhooks/{webhook_id}:generateReceiverCode supports.
    """

    targets: list[ReceiverTargetOut]

    def to_dict(self) -> dict[str, Any]:
        targets = []
        for targets_item_data in self.targets:
            targets_item = targets_item_data.to_dict()
            targets.append(targets_item)

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "targets": targets,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.receiver_target_out import ReceiverTargetOut  # noqa: PLC0415

        d = dict(src_dict)
        targets = []
        _targets = d.pop("targets")
        for targets_item_data in _targets:
            targets_item = ReceiverTargetOut.from_dict(targets_item_data)

            targets.append(targets_item)

        receiver_code_struct = cls(
            targets=targets,
        )

        return receiver_code_struct
