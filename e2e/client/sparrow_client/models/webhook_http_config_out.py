from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="WebhookHTTPConfigOut")


@_attrs_define
class WebhookHTTPConfigOut:
    """
    Attributes:
        capture_response_body (bool): false: up to 1 KiB of each response body is stored; true: up to
            SPARROW_MAX_CAPTURED_RESPONSE_BYTES (default 1 MiB).
        follow_redirects (bool): Whether HTTP redirects returned by the endpoint are followed.
        max_retries (int): Retries after the first attempt before a delivery is marked failed (up to max_retries + 1
            attempts).
        request_timeout_seconds (int): Per-attempt request timeout, in seconds.
        retry_backoff_seconds (int): Base retry delay, in seconds; the delay after attempt N is retry_backoff_seconds *
            2^(N-1), capped at 24 hours.
        verify_ssl (bool): Whether the endpoint's TLS certificate is verified.
        content_type (str | Unset): Stored value; deliveries currently send Content-Type: application/json unless
            headers override it.
        expected_status_codes (list[int] | None | Unset): HTTP status codes treated as a successful delivery.
        rate_limit_rps (float | Unset): Delivery rate limit for this webhook, in requests per second; absent means no
            limit.
        user_agent (str | Unset): Stored value; deliveries currently always send User-Agent: Sparrow-Webhook/<server
            version> unless headers override it.
        webhook_secret (str | Unset): HMAC signing secret. Shown in full only once, in the response to webhook
            registration; masked on every later read.
    """

    capture_response_body: bool
    follow_redirects: bool
    max_retries: int
    request_timeout_seconds: int
    retry_backoff_seconds: int
    verify_ssl: bool
    content_type: str | Unset = UNSET
    expected_status_codes: list[int] | None | Unset = UNSET
    rate_limit_rps: float | Unset = UNSET
    user_agent: str | Unset = UNSET
    webhook_secret: str | Unset = UNSET

    def to_dict(self) -> dict[str, Any]:
        capture_response_body = self.capture_response_body

        follow_redirects = self.follow_redirects

        max_retries = self.max_retries

        request_timeout_seconds = self.request_timeout_seconds

        retry_backoff_seconds = self.retry_backoff_seconds

        verify_ssl = self.verify_ssl

        content_type = self.content_type

        expected_status_codes: list[int] | None | Unset
        if isinstance(self.expected_status_codes, Unset):
            expected_status_codes = UNSET
        elif isinstance(self.expected_status_codes, list):
            expected_status_codes = self.expected_status_codes

        else:
            expected_status_codes = self.expected_status_codes

        rate_limit_rps = self.rate_limit_rps

        user_agent = self.user_agent

        webhook_secret = self.webhook_secret

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "capture_response_body": capture_response_body,
                "follow_redirects": follow_redirects,
                "max_retries": max_retries,
                "request_timeout_seconds": request_timeout_seconds,
                "retry_backoff_seconds": retry_backoff_seconds,
                "verify_ssl": verify_ssl,
            }
        )
        if content_type is not UNSET:
            field_dict["content_type"] = content_type
        if expected_status_codes is not UNSET:
            field_dict["expected_status_codes"] = expected_status_codes
        if rate_limit_rps is not UNSET:
            field_dict["rate_limit_rps"] = rate_limit_rps
        if user_agent is not UNSET:
            field_dict["user_agent"] = user_agent
        if webhook_secret is not UNSET:
            field_dict["webhook_secret"] = webhook_secret

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        capture_response_body = d.pop("capture_response_body")

        follow_redirects = d.pop("follow_redirects")

        max_retries = d.pop("max_retries")

        request_timeout_seconds = d.pop("request_timeout_seconds")

        retry_backoff_seconds = d.pop("retry_backoff_seconds")

        verify_ssl = d.pop("verify_ssl")

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

        rate_limit_rps = d.pop("rate_limit_rps", UNSET)

        user_agent = d.pop("user_agent", UNSET)

        webhook_secret = d.pop("webhook_secret", UNSET)

        webhook_http_config_out = cls(
            capture_response_body=capture_response_body,
            follow_redirects=follow_redirects,
            max_retries=max_retries,
            request_timeout_seconds=request_timeout_seconds,
            retry_backoff_seconds=retry_backoff_seconds,
            verify_ssl=verify_ssl,
            content_type=content_type,
            expected_status_codes=expected_status_codes,
            rate_limit_rps=rate_limit_rps,
            user_agent=user_agent,
            webhook_secret=webhook_secret,
        )

        return webhook_http_config_out
