#!/usr/bin/env python3
"""
Push local markdown learnings to ALMS using direct MCP JSON-RPC calls.

This script is designed to be the primary publish path for agents that keep a
local `learnings/` directory. It performs a targeted ALMS dedup lookup before
storing each learning and supports dry-run mode by default.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import urllib.error
from pathlib import Path
from typing import Any

from alms_mcp import ALMSMCPClient

ALMS_URL = os.getenv("ALMS_URL", "http://localhost:8001/mcp")
ALMS_TOKEN = os.getenv("ALMS_AUTH_TOKEN", "")
AGENT_ID = os.getenv("AGENT_ID", "openclaw-orch")
LEARNINGS_DIR = Path(
    os.getenv(
        "LEARNINGS_DIR",
        str(Path(__file__).resolve().parent.parent / "learnings"),
    )
)

VALID_TYPES = {"pattern", "failure", "config", "protocol", "edge_case"}
LEARNING_TYPE_PATTERNS = {
    "lessons": "pattern",
    "decision": "pattern",
    "methodology": "protocol",
    "config": "config",
    "failure": "failure",
    "protocol": "protocol",
    "edge": "edge_case",
    "design": "pattern",
}


def parse_args() -> argparse.Namespace:
    """Parse CLI flags."""
    parser = argparse.ArgumentParser(
        description="Push local markdown learnings to ALMS.",
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Actually call learning.store. Default is dry-run.",
    )
    parser.add_argument(
        "--agent-id",
        default=AGENT_ID,
        help=f"Agent identifier (default: {AGENT_ID}).",
    )
    parser.add_argument(
        "--dir",
        dest="learnings_dir",
        default=str(LEARNINGS_DIR),
        help=f"Directory containing markdown learnings (default: {LEARNINGS_DIR}).",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=0,
        help="Only process the first N extracted learnings.",
    )
    return parser.parse_args()


def extract_tool_payload(response: Any) -> Any:
    """Extract the JSON payload returned by mcp.NewToolResultText."""
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
        content = result.get("content")
        if isinstance(content, list) and content:
            first = content[0]
            if isinstance(first, dict) and "text" in first:
                return json.loads(first["text"])

    return result


def infer_learning_type(filename: str, content: str) -> str:
    """Infer a learning type from the filename and markdown body."""
    stem = filename.lower().replace(" ", "-")
    learning_type = "pattern"

    for pattern, mapped_type in LEARNING_TYPE_PATTERNS.items():
        if pattern in stem:
            learning_type = mapped_type
            break

    type_match = re.search(r"\*\*Type:\*\*\s*`?([\w_]+)`?", content)
    if type_match:
        candidate = type_match.group(1).lower()
        if candidate in VALID_TYPES:
            return candidate

    return learning_type


def extract_tags(filename: str, content: str, agent_id: str) -> list[str]:
    """Extract tags from markdown and add stable defaults."""
    stem = Path(filename).stem.lower()
    defaults = [agent_id]
    if "-" in stem:
        defaults.insert(0, stem.split("-")[0])
    else:
        defaults.insert(0, stem)

    tag_match = re.search(r"\*\*Tags:\*\*\s*(.+?)(?:\n|$)", content)
    if tag_match:
        extras = [tag.strip().strip("`#") for tag in tag_match.group(1).split(",")]
        defaults.extend(tag for tag in extras if tag)

    unique_tags: list[str] = []
    seen: set[str] = set()
    for tag in defaults:
        normalized = tag.strip().lower().replace(" ", "_")
        if normalized and normalized not in seen:
            seen.add(normalized)
            unique_tags.append(normalized)

    return unique_tags


def extract_body(content: str) -> str:
    """Extract the markdown body excluding metadata lines."""
    lines: list[str] = []
    in_frontmatter = False

    for raw_line in content.splitlines():
        line = raw_line.rstrip()
        if line == "---":
            in_frontmatter = not in_frontmatter
            continue
        if in_frontmatter:
            continue
        if line.startswith("# "):
            continue
        if line.startswith("**Type:**") or line.startswith("**Source:**"):
            continue
        if line.startswith("**Created:**") or line.startswith("**Tags:**"):
            continue
        if line.startswith("**ALMS ID:**"):
            continue
        lines.append(raw_line)

    return "\n".join(lines).strip()


def extract_learnings(learnings_dir: Path, agent_id: str) -> list[dict[str, Any]]:
    """Extract learnings from local markdown files."""
    if not learnings_dir.is_dir():
        print(f"[ERROR] learnings dir not found: {learnings_dir}")
        return []

    learnings: list[dict[str, Any]] = []

    for path in sorted(learnings_dir.iterdir()):
        if path.suffix != ".md" or path.name == "README.md":
            continue

        content = path.read_text(encoding="utf-8")
        heading_match = re.search(r"^#\s+(.+)$", content, re.MULTILINE)
        title = heading_match.group(1).strip() if heading_match else path.stem
        body = extract_body(content)
        if not body:
            continue

        learnings.append(
            {
                "title": title,
                "body": body[:4000],
                "type": infer_learning_type(path.name, content),
                "tags": extract_tags(path.name, content, agent_id),
                "source_path": str(path),
            }
        )

    return learnings


def find_existing_learning(
    client: ALMSMCPClient,
    title: str,
    agent_id: str,
) -> dict[str, Any] | None:
    """Search ALMS for an exact-title learning from the same source agent."""
    payload = client.call_tool(
        "learning.search",
        {
            "query": title,
            "limit": 10,
            "status": "all",
            "include_rejected": True,
        },
    )
    if not isinstance(payload, list):
        raise RuntimeError(f"expected search results list, got {type(payload).__name__}")

    normalized_title = title.strip().casefold()
    for item in payload:
        if not isinstance(item, dict):
            continue
        existing_title = str(item.get("title", "")).strip().casefold()
        existing_agent = str(item.get("src_agent_id", ""))
        if existing_title == normalized_title and existing_agent == agent_id:
            return item

    return None


def push_learning(
    client: ALMSMCPClient,
    learning: dict[str, Any],
    agent_id: str,
    apply: bool,
) -> bool:
    """Push a single learning to ALMS or report a dry-run result."""
    if not apply:
        print(f"  [DRY] {learning['type']}: {learning['title']}")
        return True

    payload = client.call_tool(
        "learning.store",
        {
            "agent_id": agent_id,
            "title": learning["title"],
            "type": learning["type"],
            "tags": learning["tags"],
            "body": learning["body"],
        },
    )
    if not isinstance(payload, dict):
        raise RuntimeError(f"unexpected learning.store response: {payload}")

    print(
        "  [OK] "
        f"{learning['type']}: {learning['title']} "
        f"-> {payload.get('learning_id', 'unknown')} "
        f"(duplicate={payload.get('is_duplicate', False)})"
    )
    return True


def main() -> int:
    """Run the CLI."""
    args = parse_args()
    learnings_dir = Path(args.learnings_dir)

    if args.apply:
        print(f"=== Pushing learnings to ALMS ({ALMS_URL}) ===")
    else:
        print(f"=== DRY-RUN: pushing learnings to ALMS ({ALMS_URL}) ===")
        print("  Use --apply to actually push")

    print(f"  Agent ID: {args.agent_id}")
    print(f"  Learnings dir: {learnings_dir}")
    client = ALMSMCPClient(
        ALMS_URL,
        auth_token=ALMS_TOKEN,
        client_name="alms-push-local-learnings",
    )

    learnings = extract_learnings(learnings_dir, args.agent_id)
    if args.limit > 0:
        learnings = learnings[: args.limit]
    print(f"  Extracted {len(learnings)} local learnings")

    pushed = 0
    skipped = 0

    for learning in learnings:
        try:
            existing = find_existing_learning(client, learning["title"], args.agent_id)
        except Exception as exc:  # pylint: disable=broad-except
            print(f"  [FAIL] dedup lookup for {learning['title']}: {exc}")
            return 1

        if existing is not None:
            skipped += 1
            print(
                "  [SKIP] "
                f"{learning['title']} already exists as {existing.get('learning_id', 'unknown')}"
            )
            continue

        try:
            if push_learning(client, learning, args.agent_id, args.apply):
                pushed += 1
        except Exception as exc:  # pylint: disable=broad-except
            print(f"  [FAIL] {learning['title']}: {exc}")
            return 1

    print(f"\n=== Summary: {pushed} pushed, {skipped} skipped by dedup ===")
    return 0


if __name__ == "__main__":
    sys.exit(main())
