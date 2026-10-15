from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define

from ..models.ai_drafting_struct_provider import AIDraftingStructProvider
from ..types import UNSET, Unset

T = TypeVar("T", bound="AIDraftingStruct")


@_attrs_define
class AIDraftingStruct:
    """
    Attributes:
        enabled (bool): True when the server is started with SPARROW_AI_ENABLED=true, whether or not the provider
            settings are valid. Drafting (POST /v1/subscriptions:draftTemplate) is usable only when enabled is true and
            problem is absent.
        prompt_only (bool): Always the opposite of enabled: true when SPARROW_AI_ENABLED is not true, so clients offer
            only the copy-and-paste prompt. POST /v1/subscriptions:draftTemplatePrompt works whatever this says.
        model (str | Unset): Model id used for drafting; absent when drafting is off or not usable.
        problem (str | Unset): Why drafting is enabled but not usable: a missing or invalid provider setting. POST
            /v1/subscriptions:draftTemplate answers 503 with this text.
        provider (AIDraftingStructProvider | Unset): Chat API behind drafting: anthropic, or openai for any OpenAI-
            compatible server (Ollama, vLLM, OpenRouter, OpenAI). Absent when drafting is off or not usable.
    """

    enabled: bool
    prompt_only: bool
    model: str | Unset = UNSET
    problem: str | Unset = UNSET
    provider: AIDraftingStructProvider | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        enabled = self.enabled

        prompt_only = self.prompt_only

        model = self.model

        problem = self.problem

        provider: str | Unset = UNSET
        if not isinstance(self.provider, Unset):
            provider = self.provider.value

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "enabled": enabled,
                "prompt_only": prompt_only,
            }
        )
        if model is not UNSET:
            field_dict["model"] = model
        if problem is not UNSET:
            field_dict["problem"] = problem
        if provider is not UNSET:
            field_dict["provider"] = provider

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        enabled = d.pop("enabled")

        prompt_only = d.pop("prompt_only")

        model = d.pop("model", UNSET)

        problem = d.pop("problem", UNSET)

        _provider = d.pop("provider", UNSET)
        provider: AIDraftingStructProvider | Unset
        if isinstance(_provider, Unset):
            provider = UNSET
        else:
            provider = AIDraftingStructProvider(_provider)

        ai_drafting_struct = cls(
            enabled=enabled,
            prompt_only=prompt_only,
            model=model,
            problem=problem,
            provider=provider,
        )

        return ai_drafting_struct
