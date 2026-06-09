#!/usr/bin/env python3
"""Helpers for calling ALMS over MCP Streamable HTTP."""

from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
import uuid
from typing import Any

DEFAULT_PROTOCOL_VERSION = "2025-03-26"
SESSION_HEADER = "Mcp-Session-Id"


class ALMSMCPClient:
    """Small MCP client supporting Streamable HTTP sessions and legacy fallback."""

    def __init__(
        self,
        base_url: str,
        auth_token: str = "",
        client_name: str = "alms-script",
        client_version: str = "0.1.0",
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.auth_token = auth_token
        self.client_name = client_name
        self.client_version = client_version
        self.session_id = ""
        self.protocol_version = DEFAULT_PROTOCOL_VERSION
        self.legacy_mode = False
        self.initialized = False

    def build_headers(self, include_session: bool = False) -> dict[str, str]:
        """Build base HTTP headers."""
        headers = {"Content-Type": "application/json"}
        if self.auth_token:
            headers["X-ALMS-TOKEN"] = self.auth_token
        if include_session and self.session_id:
            headers[SESSION_HEADER] = self.session_id
        return headers

    def endpoint(self, suffix: str = "") -> str:
        """Build an endpoint path relative to the MCP base URL."""
        if not suffix:
            return self.base_url
        return f"{self.base_url}/{suffix.lstrip('/')}"

    def extract_payload(self, response: Any) -> Any:
        """Extract a JSON payload from a standard MCP tool result."""
        if not isinstance(response, dict):
            return response

        if "error" in response:
            error = response["error"]
            if isinstance(error, dict):
                message = error.get("message", "unknown MCP error")
            else:
                message = str(error)
            raise RuntimeError(f"ALMS returned error: {message}")

        result = response.get("result", response)
        if isinstance(result, dict):
            if result.get("isError") is True:
                content = result.get("content")
                if isinstance(content, list) and content:
                    first = content[0]
                    if isinstance(first, dict) and "text" in first:
                        raise RuntimeError(str(first["text"]))
                raise RuntimeError("ALMS returned tool error")
            content = result.get("content")
            if isinstance(content, list) and content:
                first = content[0]
                if isinstance(first, dict) and "text" in first:
                    text = first["text"]
                    try:
                        return json.loads(text)
                    except json.JSONDecodeError:
                        return text

        return result

    def post(
        self,
        url: str,
        body: dict[str, Any],
        headers: dict[str, str],
    ) -> tuple[Any, dict[str, str]]:
        """POST a JSON-RPC request and return the parsed body and response headers."""
        request = urllib.request.Request(
            url,
            data=json.dumps(body).encode("utf-8"),
            headers=headers,
            method="POST",
        )

        try:
            with urllib.request.urlopen(request, timeout=20) as response:
                raw_bytes = response.read()
                if not raw_bytes.strip():
                    return {}, dict(response.headers.items())
                raw = json.loads(raw_bytes.decode("utf-8"))
                return raw, dict(response.headers.items())
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")
            raise RuntimeError(f"ALMS HTTP error {exc.code}: {detail}") from exc
        except urllib.error.URLError as exc:
            raise RuntimeError(f"ALMS connection failed: {exc}") from exc

    def initialize(self) -> None:
        """Initialize a Streamable HTTP MCP session, falling back to legacy mode."""
        if self.initialized:
            return

        request_body = {
            "jsonrpc": "2.0",
            "id": str(uuid.uuid4()),
            "method": "initialize",
            "params": {
                "protocolVersion": self.protocol_version,
                "clientInfo": {
                    "name": self.client_name,
                    "version": self.client_version,
                },
                "capabilities": {},
            },
        }

        try:
            response, headers = self.post(
                self.endpoint("initialize"),
                request_body,
                self.build_headers(),
            )
        except RuntimeError:
            self.legacy_mode = True
            self.initialized = True
            return

        self.session_id = headers.get(SESSION_HEADER, "")
        payload = self.extract_payload(response)
        if isinstance(payload, dict):
            self.protocol_version = payload.get("protocolVersion", self.protocol_version)

        notification = {
            "jsonrpc": "2.0",
            "method": "notifications/initialized",
        }
        self.post(self.endpoint(), notification, self.build_headers(include_session=True))

        self.initialized = True

    def call_tool(self, name: str, arguments: dict[str, Any]) -> Any:
        """Call an ALMS MCP tool."""
        self.initialize()
        body = {
            "jsonrpc": "2.0",
            "id": str(uuid.uuid4()),
            "method": "tools/call",
            "params": {
                "name": name,
                "arguments": arguments,
            },
        }

        if self.legacy_mode:
            response, _ = self.post(self.endpoint(), body, self.build_headers())
        else:
            response, _ = self.post(
                self.endpoint("tools/call"),
                body,
                self.build_headers(include_session=True),
            )

        return self.extract_payload(response)
