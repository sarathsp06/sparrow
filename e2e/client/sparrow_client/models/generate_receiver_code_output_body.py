from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.generate_receiver_code_output_body_spec import GenerateReceiverCodeOutputBodySpec
    from ..models.receiver_file_out import ReceiverFileOut


T = TypeVar("T", bound="GenerateReceiverCodeOutputBody")


@_attrs_define
class GenerateReceiverCodeOutputBody:
    """
    Attributes:
        files (list[ReceiverFileOut]): The project's files, sorted by path.
        run (str): Shell command that starts the receiver, from the project root.
        spec (GenerateReceiverCodeOutputBodySpec): Machine-readable receiver spec: how deliveries are verified and
            routed, each handler's body format and model, and the signed test vectors the generated tests replay (signed
            with throwaway credentials, never the webhook's secret). Its format is versioned by spec_version and, like the
            generated code, experimental.
        test (str): Shell command that runs the generated tests, from the project root.
        schema (str | Unset): A URL to the JSON Schema for this object.
    """

    files: list[ReceiverFileOut]
    run: str
    spec: GenerateReceiverCodeOutputBodySpec
    test: str
    schema: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        files = []
        for files_item_data in self.files:
            files_item = files_item_data.to_dict()
            files.append(files_item)

        run = self.run

        spec = self.spec.to_dict()

        test = self.test

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "files": files,
                "run": run,
                "spec": spec,
                "test": test,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.generate_receiver_code_output_body_spec import GenerateReceiverCodeOutputBodySpec  # noqa: PLC0415
        from ..models.receiver_file_out import ReceiverFileOut  # noqa: PLC0415

        d = dict(src_dict)
        files = []
        _files = d.pop("files")
        for files_item_data in _files:
            files_item = ReceiverFileOut.from_dict(files_item_data)

            files.append(files_item)

        run = d.pop("run")

        spec = GenerateReceiverCodeOutputBodySpec.from_dict(d.pop("spec"))

        test = d.pop("test")

        schema = d.pop("$schema", UNSET)

        generate_receiver_code_output_body = cls(
            files=files,
            run=run,
            spec=spec,
            test=test,
            schema=schema,
        )

        return generate_receiver_code_output_body
