import subprocess
import re
import uuid
import logging
import time

# Module-level logger
logger = logging.getLogger(__name__)

from typing import Dict, Any, List
import json


class MycelianMemoryClient:
    """CLI-based client for the Mycelian memory service used by the benchmark harness.

    Uses CLI commands as the single authoritative interface to the backend.
    See CODING_STANDARDS.md for interface policy.
    """

    def __init__(self, base_url: str, user_id: str | None = None):
        """Create client.

        If *user_id* is None, tries to use the default 'local_user' first.
        If local_user doesn't exist, creates a fresh user via CLI.
        """
        self.base_url = base_url.rstrip("/")
        
        if user_id:
            self.user_id = user_id
        else:
            self.user_id = self._get_or_create_user()

    # Regex patterns for parsing CLI output
    _VAULT_REGEX = re.compile(r"Vault created: ([a-f0-9\-]+)")
    _MEM_REGEX = re.compile(r"Memory created: ([a-f0-9\-]+)")
    _USER_REGEX = re.compile(r"User created: ([a-zA-Z0-9_\-]+)")

    def _get_or_create_user(self) -> str:
        """Try to use local_user first, create fresh user if needed."""
        # First try to use the existing local_user
        try:
            logger.info("Using existing local_user for benchmark")
            return "local_user"
        except Exception:
            # Create a fresh user
            logger.info("Creating fresh user for benchmark")
            return self._create_fresh_user()

    def _create_fresh_user(self) -> str:
        """Create a fresh benchmark user via CLI."""
        user_id = f"bench_user_{int(time.time())}"
        email = f"{user_id}@benchmark.local"
        
        out = self._run_cli(
            "create-user",
            "--user-id", user_id,
            "--email", email,
            "--display-name", "Benchmark User"
        )
        
        m = self._USER_REGEX.search(out)
        if not m:
            raise RuntimeError(f"Failed to parse user ID from CLI output:\n{out}")
        return m.group(1)

    def _run_cli(self, *args: str) -> str:
        """Run mycelianCli command and return stdout."""
        cmd = ["mycelianCli", "--service-url", self.base_url] + list(args)
        logger.debug("[CLI] %s", " ".join(cmd))
        
        try:
            result = subprocess.run(
                cmd, 
                capture_output=True, 
                text=True, 
                timeout=30,
                check=True
            )
            return result.stdout
        except subprocess.CalledProcessError as e:
            logger.warning("[CLI STDERR] %s", e.stderr)
            raise RuntimeError(
                f"mycelianCli failed (exit {e.returncode}): {' '.join(cmd)}\n"
                f"STDOUT:\n{e.stdout}\n"
                f"STDERR:\n{e.stderr}"
            ) from e

    def generate_unique_vault_title(self, prefix: str = "vault") -> str:
        """Generate a unique vault title."""
        timestamp = int(time.time())
        return f"{prefix}-run-{timestamp:08d}"

    def create_vault(self, title: str, description: str = "") -> str:
        """Create vault via CLI."""
        # Validate title meets vault constraints
        if not self._is_valid_vault_title(title):
            raise ValueError(f"Invalid vault title '{title}': must be 1-50 chars, ASCII letters/digits/hyphens only")
        
        out = self._run_cli(
            "create-vault",
            "--user-id", self.user_id,
            "--title", title,
            "--description", description,
        )
        m = self._VAULT_REGEX.search(out)
        if not m:
            raise RuntimeError(f"Failed to parse vault ID from CLI output:\n{out}")
        return m.group(1)

    def _is_valid_vault_title(self, title: str) -> bool:
        """Check if vault title meets constraints: 1-50 chars, ASCII letters/digits/hyphens only."""
        if not title or len(title) > 50:
            return False
        # Match the regex from server validation: ^[A-Za-z0-9\-]+$
        return re.match(r'^[A-Za-z0-9\-]+$', title) is not None

    def create_memory(self, title: str, memory_type: str = "chat", description: str = "", vault_id: str | None = None) -> str:
        """Create memory via CLI."""
        # If no vault provided, create a unique vault for this memory
        if vault_id is None:
            vault_title = self.generate_unique_vault_title("memory")
            vault_id = self.create_vault(vault_title, f"Auto-created vault for memory: {title}")
            logger.info("Created unique vault %s (%s) for memory %s", vault_id, vault_title, title)
            
        out = self._run_cli(
            "create-memory",
            "--user-id", self.user_id,
            "--vault-id", vault_id,
            "--title", title,
            "--memory-type", memory_type,
            "--description", description,
        )
        m = self._MEM_REGEX.search(out)
        if not m:
            raise RuntimeError(f"Failed to parse memory ID from CLI output:\n{out}")
        return m.group(1)

    def get_context(self, memory_id: str) -> str:
        """Return latest context document using CLI."""
        out = self._run_cli(
            "get-context",
            "--user-id", self.user_id,
            "--memory-id", memory_id,
        ).strip()

        if not out or out.startswith("(empty"):
            return ""

        # Attempt to parse JSON first
        try:
            data = json.loads(out)
            if isinstance(data, dict) and "content" in data:
                return data["content"]
            elif isinstance(data, str):
                return data
            else:
                return str(data)
        except json.JSONDecodeError:
            # Raw text response – return as is
            return out

    def list_entries(self, memory_id: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Return recent entries using CLI."""
        out = self._run_cli(
            "list-entries",
            "--user-id", self.user_id,
            "--memory-id", memory_id,
            "--limit", str(limit),
        ).strip()

        if not out:
            return []

        try:
            data = json.loads(out)
            if isinstance(data, list):
                return data
            elif isinstance(data, dict) and "entries" in data:
                return data["entries"]
            else:
                return []
        except json.JSONDecodeError:
            logger.warning("Failed to parse entries JSON from CLI output")
            return []

    def put_context(self, memory_id: str, context: str) -> None:
        """Update context document using CLI."""
        self._run_cli(
            "put-context",
            "--user-id", self.user_id,
            "--memory-id", memory_id,
            "--context", context,
        )

    def add_entry(self, memory_id: str, raw_entry: str, summary: str = "") -> Dict[str, Any]:
        """Add entry using CLI."""
        out = self._run_cli(
            "create-entry",
            "--user-id", self.user_id,
            "--memory-id", memory_id,
            "--raw-entry", raw_entry,
            "--summary", summary,
        )
        # CLI typically just says "Entry enqueued" - return acknowledgment
        return {"status": "enqueued", "message": out.strip()}

    def search_memories(self, memory_id: str, query: str, top_k: int = 5) -> Dict[str, Any]:
        """Search memories using CLI."""
        out = self._run_cli(
            "search",
            "--user-id", self.user_id,
            "--memory-id", memory_id,
            "--query", query,
            "--top-k", str(top_k),
        ).strip()

        try:
            return json.loads(out) if out else {"entries": []}
        except json.JSONDecodeError:
            logger.warning("Failed to parse search JSON from CLI output")
            return {"entries": []}

    def get_asset(self, asset_id: str) -> str:
        """Get asset content using CLI."""
        out = self._run_cli(
            "get-asset",
            "--id", asset_id,
        )
        return out.strip()

    def list_assets(self) -> List[str]:
        """Return list of available asset IDs via CLI."""
        try:
            out = self._run_cli("list-assets").strip()
            if not out:
                return []
            # Assume newline-separated output
            return [line.strip() for line in out.split('\n') if line.strip()]
        except Exception:
            # Return empty list if command fails
            return []

    def await_consistency(self, memory_id: str) -> None:
        """Block until previous writes for the memory are consistent using CLI."""
        try:
            self._run_cli(
                "await-consistency",
                "--user-id", self.user_id,
                "--memory-id", memory_id,
            )
        except Exception:
            # Small delay as fallback
            time.sleep(0.5)

    def get_user(self, user_id: str | None = None) -> Dict[str, Any]:
        """Get user details using CLI."""
        target_user = user_id or self.user_id
        out = self._run_cli(
            "get-user",
            "--user-id", target_user,
        ).strip()

        try:
            return json.loads(out) if out else {}
        except json.JSONDecodeError:
            logger.warning("Failed to parse user JSON from CLI output")
            return {}