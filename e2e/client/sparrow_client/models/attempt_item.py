from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..models.attempt_item_error_category import AttemptItemErrorCategory
from ..types import UNSET, Unset

T = TypeVar("T", bound="AttemptItem")


@_attrs_define
class AttemptItem:
    """
    Attributes:
        response_code (int): HTTP status code returned by the endpoint on this attempt.
        response_time (int): Round-trip time of this attempt, in milliseconds.
        success (bool): Whether this attempt succeeded (matched an expected status code).
        timestamp (str): When this attempt was made, RFC3339.
        error_category (AttemptItemErrorCategory | Unset): Failure classification for this attempt.
        error_message (str | Unset): Human-readable failure reason for this attempt.
    """

    response_code: int
    response_time: int
    success: bool
    timestamp: str
    error_category: AttemptItemErrorCategory | Unset = UNSET
    error_message: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        response_code = self.response_code

        response_time = self.response_time

        success = self.success

        timestamp = self.timestamp

        error_category: str | Unset = UNSET
        if not isinstance(self.error_category, Unset):
            error_category = self.error_category.value

        error_message = self.error_message

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "response_code": response_code,
                "response_time": response_time,
                "success": success,
                "timestamp": timestamp,
            }
        )
        if error_category is not UNSET:
            field_dict["error_category"] = error_category
        if error_message is not UNSET:
            field_dict["error_message"] = error_message

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        response_code = d.pop("response_code")

        response_time = d.pop("response_time")

        success = d.pop("success")

        timestamp = d.pop("timestamp")

        _error_category = d.pop("error_category", UNSET)
        error_category: AttemptItemErrorCategory | Unset
        if isinstance(_error_category, Unset):
            error_category = UNSET
        else:
            error_category = AttemptItemErrorCategory(_error_category)

        error_message = d.pop("error_message", UNSET)

        attempt_item = cls(
            response_code=response_code,
            response_time=response_time,
            success=success,
            timestamp=timestamp,
            error_category=error_category,
            error_message=error_message,
        )

        return attempt_item
