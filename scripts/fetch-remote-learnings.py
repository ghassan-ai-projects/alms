#!/usr/bin/env python3
"""
Fetch remote learnings from ALMS and ingest them into a local learnings folder.

Default behavior calls ALMS directly via JSON-RPC:
  1. learning.sync
  2. write remote learnings to local markdown files
  3. optionally learning.sync_ack
  4. update local cursor after a successful ack

The script can also read a saved MCP response from stdin or a file for offline
inspection, but direct MCP calls are the preferred integration path for agents.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import urllib.error
from datetime import datetime, timezone
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
CURSOR_FILE = LEARNINGS_DIR / ".cursor.json"
DEFAULT_CURSOR_TS = "1970-01-01T00:00:00Z"


def parse_args() -> argparse.Namespace:
    """Parse CLI flags."""
    parser = argparse.ArgumentParser(
        description="Fetch remote learnings from ALMS and ingest them locally.",
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Write markdown files and update cursor state.",
    )
    parser.add_argument(
        "--stdin",
        action="store_true",
        help="Read a saved JSON or MCP response from stdin instead of calling ALMS.",
    )
    parser.add_argument(
        "--from",
        dest="from_file",
        help="Read a saved JSON or MCP response from a file instead of calling ALMS.",
    )
    parser.add_argument(
        "--search-query",
        help="Use learning.search instead of learning.sync for targeted retrieval.",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=100,
        help="Maximum results for search mode (default: 100).",
    )
    parser.add_argument(
        "--type",
        dest="learning_type",
        default="",
        help="Optional learning type filter.",
    )
    parser.add_argument(
        "--tag",
        action="append",
        default=[],
        help="Optional tag filter. Repeatable.",
    )
    parser.add_argument(
        "--since",
        help="Override sync cursor timestamp. Defaults to local cursor file.",
    )
    parser.add_argument(
        "--agent-id",
        default=AGENT_ID,
        help=f"Agent identifier (default: {AGENT_ID}).",
    )
    parser.add_argument(
        "--no-ack",
        action="store_true",
        help="Skip learning.sync_ack after a direct sync call.",
    )
    return parser.parse_args()


def parse_timestamp(ts_str: str) -> datetime:
    """Parse an ISO 8601 timestamp into an aware datetime."""
    normalized = ts_str.replace("Z", "+00:00")
    return datetime.fromisoformat(normalized)


def load_cursor() -> tuple[str, str]:
    """Load the local cursor state."""
    if not CURSOR_FILE.exists():
        return "", DEFAULT_CURSOR_TS

    with CURSOR_FILE.open(encoding="utf-8") as handle:
        data = json.load(handle)
    return data.get("last_learning_id", ""), data.get("last_timestamp", DEFAULT_CURSOR_TS)


def count_markdown_files() -> int:
    """Count markdown learning files excluding README-style files."""
    if not LEARNINGS_DIR.is_dir():
        return 0

    count = 0
    for path in LEARNINGS_DIR.iterdir():
        if path.suffix == ".md" and path.name != "README.md":
            count += 1
    return count


def save_cursor(learning_id: str, timestamp: str) -> None:
    """Persist the local cursor."""
    LEARNINGS_DIR.mkdir(parents=True, exist_ok=True)
    payload = {
        "last_learning_id": learning_id,
        "last_timestamp": timestamp,
        "ingested_count": count_markdown_files(),
        "ingested_at": datetime.now(timezone.utc).isoformat(),
    }
    with CURSOR_FILE.open("w", encoding="utf-8") as handle:
        json.dump(payload, handle, indent=2)


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


def load_external_payload(args: argparse.Namespace) -> Any:
    """Load a saved payload from stdin or a file."""
    if args.stdin:
        raw = sys.stdin.read()
    elif args.from_file:
        raw = Path(args.from_file).read_text(encoding="utf-8")
    else:
        raise ValueError("no external payload source provided")

    return extract_tool_payload(json.loads(raw))


def load_learnings(
    client: ALMSMCPClient,
    args: argparse.Namespace,
    since: str,
) -> tuple[list[dict[str, Any]], bool]:
    """Load learnings either from ALMS directly or from an external payload."""
    if args.stdin or args.from_file:
        payload = load_external_payload(args)
        if not isinstance(payload, list):
            raise RuntimeError(f"expected a list of learnings, got {type(payload).__name__}")
        return payload, False

    if args.search_query:
        payload = client.call_tool(
            "learning.search",
            {
                "query": args.search_query,
                "type": args.learning_type,
                "tags": args.tag,
                "limit": args.limit,
                "status": "all",
                "include_rejected": False,
            },
        )
        if not isinstance(payload, list):
            raise RuntimeError(f"expected search results list, got {type(payload).__name__}")
        return payload, False

    payload = client.call_tool(
        "learning.sync",
        {
            "agent_id": args.agent_id,
            "since": since,
            "type": args.learning_type,
            "tags": args.tag,
        },
    )
    if not isinstance(payload, list):
        raise RuntimeError(f"expected sync results list, got {type(payload).__name__}")
    return payload, True


def filter_remote_learnings(
    learnings: list[dict[str, Any]],
    agent_id: str,
    since_ts: str,
) -> list[dict[str, Any]]:
    """Keep only remote learnings newer than the cursor."""
    cutoff = (
        parse_timestamp(since_ts)
        if since_ts and since_ts != DEFAULT_CURSOR_TS
        else datetime.min.replace(tzinfo=timezone.utc)
    )

    remote: list[dict[str, Any]] = []
    seen_ids: set[str] = set()

    for learning in learnings:
        if not isinstance(learning, dict):
            continue
        if learning.get("src_agent_id") == agent_id:
            continue

        learning_id = str(learning.get("learning_id", ""))
        if learning_id and learning_id in seen_ids:
            continue
        if learning_id:
            seen_ids.add(learning_id)

        created_at = str(learning.get("created_at", ""))
        if not created_at:
            remote.append(learning)
            continue

        try:
            created = parse_timestamp(created_at)
        except ValueError:
            remote.append(learning)
            continue

        if created > cutoff:
            remote.append(learning)

    remote.sort(key=lambda item: (str(item.get("created_at", "")), str(item.get("learning_id", ""))))
    return remote


def safe_filename(title: str, learning_id: str) -> str:
    """Build a stable markdown filename for a learning."""
    cleaned = re.sub(r"[^\w\s-]", "", title).strip().replace(" ", "-")
    cleaned = re.sub(r"-+", "-", cleaned)[:60] or "untitled"
    prefix = learning_id[:8] if learning_id else "unknown"
    return f"{prefix}-{cleaned}.md"


def learning_to_markdown(learning: dict[str, Any]) -> tuple[Path, str]:
    """Render a learning record as a markdown file."""
    title = str(learning.get("title", "Untitled"))
    learning_type = str(learning.get("type", "pattern"))
    author = str(learning.get("src_agent_id", "unknown"))
    body = str(learning.get("body", "")).strip()
    tags = learning.get("tags", [])
    created = str(learning.get("created_at", ""))
    learning_id = str(learning.get("learning_id", "unknown"))

    if not isinstance(tags, list):
        tags = []

    markdown = f"""# {title}

**Type:** `{learning_type}`
**Source:** {author}
**Created:** {created}
**Tags:** {", ".join(str(tag) for tag in tags[:10])}
**ALMS ID:** {learning_id}

---

{body}
"""

    return LEARNINGS_DIR / safe_filename(title, learning_id), markdown


def ingest_learnings(learnings: list[dict[str, Any]], apply: bool) -> list[dict[str, str]]:
    """Write local markdown files and return processed IDs/timestamps."""
    LEARNINGS_DIR.mkdir(parents=True, exist_ok=True)
    processed: list[dict[str, str]] = []

    for learning in learnings:
        path, markdown = learning_to_markdown(learning)
        if apply:
            path.write_text(markdown, encoding="utf-8")
            print(f"  [OK] {path.name}")
        else:
            print(f"  [DRY] {path.name}")

        processed.append(
            {
                "learning_id": str(learning.get("learning_id", "")),
                "created_at": str(learning.get("created_at", "")),
            }
        )

    return processed


def latest_cursor_entry(entries: list[dict[str, str]], current_id: str, current_ts: str) -> tuple[str, str]:
    """Return the latest cursor pair from processed entries."""
    latest_id = current_id
    latest_ts = current_ts

    for entry in entries:
        learning_id = entry["learning_id"]
        created_at = entry["created_at"]
        if created_at > latest_ts or (created_at == latest_ts and learning_id > latest_id):
            latest_id = learning_id
            latest_ts = created_at

    return latest_id, latest_ts


def build_sync_entries(learnings: list[dict[str, Any]]) -> list[dict[str, str]]:
    """Build cursor/ack metadata from the full sync batch."""
    entries: list[dict[str, str]] = []
    for learning in learnings:
        if not isinstance(learning, dict):
            continue
        entries.append(
            {
                "learning_id": str(learning.get("learning_id", "")),
                "created_at": str(learning.get("created_at", "")),
            }
        )
    return entries


def ack_sync_batch(client: ALMSMCPClient, agent_id: str, learning_ids: list[str]) -> None:
    """Acknowledge a processed sync batch."""
    if not learning_ids:
        return

    payload = client.call_tool(
        "learning.sync_ack",
        {
            "agent_id": agent_id,
            "learning_ids": learning_ids,
        },
    )
    if not isinstance(payload, dict) or payload.get("status") != "acknowledged":
        raise RuntimeError(f"unexpected sync_ack response: {payload}")


def main() -> int:
    """Run the CLI."""
    args = parse_args()
    last_id, cursor_ts = load_cursor()
    since = args.since or cursor_ts
    client = ALMSMCPClient(
        ALMS_URL,
        auth_token=ALMS_TOKEN,
        client_name="alms-fetch-remote-learnings",
    )

    mode = "sync"
    if args.stdin or args.from_file:
        mode = "external"
    elif args.search_query:
        mode = "search"

    if args.apply:
        print(f"=== Ingesting remote learnings ({mode}) -> {LEARNINGS_DIR} ===")
    else:
        print(f"=== DRY-RUN ({mode}) -> {LEARNINGS_DIR} ===")
        print("  Use --apply to write files")

    print(f"  Agent ID: {args.agent_id}")
    print(f"  Cursor: id={last_id}, ts={since}")

    try:
        learnings, is_sync = load_learnings(client, args, since)
    except Exception as exc:  # pylint: disable=broad-except
        print(f"[ERROR] {exc}")
        return 1

    print(f"  Total learnings loaded: {len(learnings)}")
    remote_learnings = filter_remote_learnings(learnings, args.agent_id, since)
    print(f"  Remote learnings after filter: {len(remote_learnings)}")

    processed_entries = ingest_learnings(remote_learnings, args.apply)
    sync_entries = build_sync_entries(learnings) if is_sync else processed_entries

    if args.apply and is_sync and not args.no_ack and sync_entries:
        learning_ids = [entry["learning_id"] for entry in sync_entries if entry["learning_id"]]
        try:
            ack_sync_batch(client, args.agent_id, learning_ids)
        except Exception as exc:  # pylint: disable=broad-except
            print(f"[ERROR] sync_ack failed: {exc}")
            return 1
        print(f"  Acked {len(learning_ids)} learning IDs")

    if args.apply and sync_entries:
        if is_sync and args.no_ack:
            print("  Cursor not updated because --no-ack was set")
        else:
            latest_id, latest_ts = latest_cursor_entry(sync_entries, last_id, since)
            save_cursor(latest_id, latest_ts)
            print(f"  Cursor updated: id={latest_id}, ts={latest_ts}")

    print(f"\n=== Summary: {len(processed_entries)} remote learnings processed ===")
    return 0


if __name__ == "__main__":
    sys.exit(main())
