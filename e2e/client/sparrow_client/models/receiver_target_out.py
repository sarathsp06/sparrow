from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="ReceiverTargetOut")


@_attrs_define
class ReceiverTargetOut:
    """
    Attributes:
        framework (str): Value for the framework field of generateReceiverCode.
        id (str): Target id, e.g. go-nethttp.
        language (str): Value for the language field of generateReceiverCode.
        name (str): Display name.
        runtime (str): Runtime the generated project needs.
    """

    framework: str
    id: str
    language: str
    name: str
    runtime: str

    def to_dict(self) -> dict[str, Any]:
        framework = self.framework

        id = self.id

        language = self.language

        name = self.name

        runtime = self.runtime

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "framework": framework,
                "id": id,
                "language": language,
                "name": name,
                "runtime": runtime,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        framework = d.pop("framework")

        id = d.pop("id")

        language = d.pop("language")

        name = d.pop("name")

        runtime = d.pop("runtime")

        receiver_target_out = cls(
            framework=framework,
            id=id,
            language=language,
            name=name,
            runtime=runtime,
        )

        return receiver_target_out
