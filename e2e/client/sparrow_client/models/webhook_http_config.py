from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="WebhookHTTPConfig")


@_attrs_define
class WebhookHTTPConfig:
    """
    Attributes:
        capture_response_body (bool | Unset): How much of the endpoint's response body is stored with each attempt:
            false (default) keeps up to 1 KiB, true up to SPARROW_MAX_CAPTURED_RESPONSE_BYTES (default 1 MiB).
        content_type (str | Unset): Stored and returned, but deliveries do not use it yet: they send Content-Type:
            application/json. To send another Content-Type, set it in headers (or the subscription's headers). Must not be
            empty; defaults to application/json.
        expected_status_codes (list[int] | None | Unset): HTTP status codes treated as a successful delivery, each
            between 100 and 599. Defaults to 200, 201, 202 and 204.
        follow_redirects (bool | Unset): Whether to follow HTTP redirects returned by the endpoint. Each redirect target
            passes the same SSRF checks. Defaults to true.
        max_retries (int | Unset): Retries after the first attempt before a delivery is marked failed, so a delivery
            gets up to max_retries + 1 attempts. 0 means no retries. Between 0 and 10; defaults to 3.
        rate_limit_rps (float | Unset): Maximum delivery rate to this webhook, in requests per second (fractions
            allowed, e.g. 0.5). Must be positive; unset means no limit. Deliveries over the limit wait in the queue.
        request_timeout_seconds (int | Unset): How long to wait for the endpoint to respond before treating the attempt
            as a timeout, in seconds. Between 1 and 300; defaults to 30.
        retry_backoff_seconds (int | Unset): Base retry delay, in seconds. The delay after attempt N is
            retry_backoff_seconds * 2^(N-1), capped at 24 hours. Between 1 and 3600; defaults to 60.
        user_agent (str | Unset): Stored and returned, but deliveries do not use it yet: they always send User-Agent:
            Sparrow-Webhook/<server version>. To send another User-Agent, set it in headers. Defaults to Sparrow-
            Webhook/1.0.
        verify_ssl (bool | Unset): Whether to verify the endpoint's TLS certificate. Defaults to true. Set false only
            for trusted internal endpoints with self-signed certificates; SSRF and redirect checks still apply.
    """

    capture_response_body: bool | Unset = UNSET
    content_type: str | Unset = UNSET
    expected_status_codes: list[int] | None | Unset = UNSET
    follow_redirects: bool | Unset = UNSET
    max_retries: int | Unset = UNSET
    rate_limit_rps: float | Unset = UNSET
    request_timeout_seconds: int | Unset = UNSET
    retry_backoff_seconds: int | Unset = UNSET
    user_agent: str | Unset = UNSET
    verify_ssl: bool | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        capture_response_body = self.capture_response_body

        content_type = self.content_type

        expected_status_codes: list[int] | None | Unset
        if isinstance(self.expected_status_codes, Unset):
            expected_status_codes = UNSET
        elif isinstance(self.expected_status_codes, list):
            expected_status_codes = self.expected_status_codes

        else:
            expected_status_codes = self.expected_status_codes

        follow_redirects = self.follow_redirects

        max_retries = self.max_retries

        rate_limit_rps = self.rate_limit_rps

        request_timeout_seconds = self.request_timeout_seconds

        retry_backoff_seconds = self.retry_backoff_seconds

        user_agent = self.user_agent

        verify_ssl = self.verify_ssl

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if capture_response_body is not UNSET:
            field_dict["capture_response_body"] = capture_response_body
        if content_type is not UNSET:
            field_dict["content_type"] = content_type
        if expected_status_codes is not UNSET:
            field_dict["expected_status_codes"] = expected_status_codes
        if follow_redirects is not UNSET:
            field_dict["follow_redirects"] = follow_redirects
        if max_retries is not UNSET:
            field_dict["max_retries"] = max_retries
        if rate_limit_rps is not UNSET:
            field_dict["rate_limit_rps"] = rate_limit_rps
        if request_timeout_seconds is not UNSET:
            field_dict["request_timeout_seconds"] = request_timeout_seconds
        if retry_backoff_seconds is not UNSET:
            field_dict["retry_backoff_seconds"] = retry_backoff_seconds
        if user_agent is not UNSET:
            field_dict["user_agent"] = user_agent
        if verify_ssl is not UNSET:
            field_dict["verify_ssl"] = verify_ssl

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        capture_response_body = d.pop("capture_response_body", UNSET)

        content_type = d.pop("content_type", UNSET)

        def _parse_expected_status_codes(data: object) -> list[int] | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                expected_status_codes_type_0 = cast(list[int], data)

                return expected_status_codes_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[int] | None | Unset, data)

        expected_status_codes = _parse_expected_status_codes(d.pop("expected_status_codes", UNSET))

        follow_redirects = d.pop("follow_redirects", UNSET)

        max_retries = d.pop("max_retries", UNSET)

        rate_limit_rps = d.pop("rate_limit_rps", UNSET)

        request_timeout_seconds = d.pop("request_timeout_seconds", UNSET)

        retry_backoff_seconds = d.pop("retry_backoff_seconds", UNSET)

        user_agent = d.pop("user_agent", UNSET)

        verify_ssl = d.pop("verify_ssl", UNSET)

        webhook_http_config = cls(
            capture_response_body=capture_response_body,
            content_type=content_type,
            expected_status_codes=expected_status_codes,
            follow_redirects=follow_redirects,
            max_retries=max_retries,
            rate_limit_rps=rate_limit_rps,
            request_timeout_seconds=request_timeout_seconds,
            retry_backoff_seconds=retry_backoff_seconds,
            user_agent=user_agent,
            verify_ssl=verify_ssl,
        )

        return webhook_http_config
