import subprocess
import re
import uuid
import logging
import time

# Module-level logger
logger = logging.getLogger(__name__)

from typing import Dict, Any, List, Optional
import json


class MycelianMemoryClient:
    """CLI-based client for the Mycelian memory service used by the benchmark harness.

    Uses CLI commands as the single authoritative interface to the backend.
    See CODING_STANDARDS.md for interface policy.
    """

    def __init__(self, base_url: str):
        """Create client for dev mode (no user management)."""
        self.base_url = base_url.rstrip("/")





    # ------------------------------------------------------------------
    # CLI helpers
    # ------------------------------------------------------------------
    _MEM_REGEX = re.compile(r"Memory created: ([0-9a-fA-F-]{36})")
    _VAULT_REGEX = re.compile(r"Vault created: ([0-9a-fA-F-]{36})")

    def _run_cli(self, *args: str) -> str:
        """Run the `mycelianCli` CLI and return stdout."""
        cmd = ["mycelianCli", "--service-url", self.base_url, *args]
        logging.info("[CLI] %s", " ".join(cmd))
        res = subprocess.run(cmd, capture_output=True, text=True, check=False)

        # Always show raw CLI output for transparency, but avoid flooding logs
        is_asset_cmd = args and args[0] == "get-asset"

        if res.stdout.strip() and not is_asset_cmd:
            # For non-asset commands, surface stdout verbatim
            print(res.stdout.strip())
        if res.stderr.strip():
            if is_asset_cmd:
                # Asset bodies can be large – log only metadata
                logging.info("[CLI STDERR] (get-asset output suppressed – %d chars)", len(res.stderr))
            elif len(res.stderr) > 500:
                logging.warning("[CLI STDERR] (truncated) %s…", res.stderr[:500])
            else:
                logging.warning("[CLI STDERR] %s", res.stderr.strip())

        if res.returncode != 0:
            raise RuntimeError(
                f"mycelianCli CLI failed (exit {res.returncode}): {' '.join(cmd)}\nSTDOUT:\n{res.stdout}\nSTDERR:\n{res.stderr}"
            )

        return res.stdout



    # ---------------------------------------------------------------------
    # Vault lifecycle  
    # ---------------------------------------------------------------------
    def create_vault(self, title: str, description: str = "") -> str:
        """Create a new vault and return its ID.
        
        Title must be 1-50 chars, ASCII letters/digits/hyphens only.
        Uses CLI to avoid schema drift with Go client.
        """
        # Validate title meets vault constraints
        if not self._is_valid_vault_title(title):
            raise ValueError(f"Invalid vault title '{title}': must be 1-50 chars, ASCII letters/digits/hyphens only")
        
        out = self._run_cli(
            "create-vault",
            "--title", 
            title,
            "--description",
            description,
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

    def generate_unique_vault_title(self, base_name: str = "benchmark") -> str:
        """Generate a unique vault title with timestamp for benchmark runs.
        
        Returns a title like 'benchmark-run-20250103-145623' that meets vault constraints.
        """
        import datetime
        timestamp = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
        # Replace any invalid characters in base_name
        clean_base = re.sub(r'[^A-Za-z0-9\-]', '-', base_name)
        title = f"{clean_base}-run-{timestamp}"
        # Ensure we don't exceed 50 char limit
        if len(title) > 50:
            # Truncate base_name to fit
            max_base_len = 50 - len(f"-run-{timestamp}")
            clean_base = clean_base[:max_base_len]
            title = f"{clean_base}-run-{timestamp}"
        return title

    # ---------------------------------------------------------------------
    # Memory lifecycle
    # ---------------------------------------------------------------------
    def create_memory(self, title: str, memory_type: str = "chat", description: str = "", vault_id: str | None = None) -> str:
        """Create a new memory and return its ID.

        If vault_id is not provided, creates a unique vault for this memory.
        Primary path uses the `mycelianCli` CLI to avoid accidental schema drift
        with the Go client.  If the CLI returns a non-zero exit code (often due
        to the backend returning HTTP 500 while still starting up), we fall
        back to a direct HTTP POST which is faster and less brittle inside the
        pytest process.  Either path must return the new ``memoryId``.
        """
        # If no vault provided, create a unique vault for this memory
        if vault_id is None:
            vault_title = self.generate_unique_vault_title("memory")
            vault_id = self.create_vault(vault_title, f"Auto-created vault for memory: {title}")
            logger.info("Created unique vault %s (%s) for memory %s", vault_id, vault_title, title)
        out = self._run_cli(
            "create-memory",
            "--vault-id",
            vault_id,
            "--title",
            title,
            "--memory-type",
            memory_type,
            "--description",
            description,
        )
        m = self._MEM_REGEX.search(out)
        if not m:
            raise RuntimeError(f"Failed to parse memory ID from CLI output:\n{out}")
        return m.group(1)

    # ---------------------------------------------------------------------
    # Read operations
    # ---------------------------------------------------------------------
    def get_context(self, memory_id: str) -> str:
        """Return latest context document using CLI."""

        try:
            out = self._run_cli(
                "get-context",
                "--memory-id",
                memory_id,
            ).strip()

            if not out or out.startswith("(empty"):
                return ""

            # Attempt to parse JSON first
            try:
                data = json.loads(out)
            except json.JSONDecodeError:
                # Raw string context was printed
                return out

            # JSON object path – newer CLI prints full struct
            if isinstance(data, dict):
                ctx_obj = data.get("context") or data.get("content") or data
                if isinstance(ctx_obj, str):
                    return ctx_obj
                return json.dumps(ctx_obj)

            # Fallback: return raw
            return out

        except Exception as cli_err:
            logger.debug("[get_context] CLI failed (%s); HTTP fallback disabled (no users)", cli_err)
            return ""

        # ------------------------------------------------------------------
        # HTTP fallback disabled - user management removed
        # ------------------------------------------------------------------

    # ------------------------------------------------------------------
    # Public read helper matching tool name exactly
    # ------------------------------------------------------------------

    def list_entries(self, memory_id: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Return recent entries using CLI; fallback to HTTP."""
        try:
            out = self._run_cli(
                "list-entries",
                "--memory-id",
                memory_id,
                "--limit",
                str(limit),
            )
            out = out.strip()
            if not out:
                return []
            data = json.loads(out)
            return data.get("entries") or []
        except Exception as cli_err:
            logger.debug("[list_entries] CLI failed (%s); HTTP fallback disabled (no users)", cli_err)
            return []

    # Backwards-compatibility alias (deprecated)
    def list_recent_entries(self, memory_id: str, limit: int = 10):  # pragma: no cover
        return self.list_entries(memory_id, limit)

    # ---------------------------------------------------------------------
    # Write operations
    # ---------------------------------------------------------------------
    def add_entry(
        self,
        memory_id: str,
        raw_entry: str,
        summary: str,
        *,
        role: str,
        tags: Optional[Dict[str, Any]] = None,
    ):
        """Persist an entry via HTTP. `role` is mandatory ("speaker 1" or "speaker 2")."""

        if not role:
            raise ValueError("add_entry requires non-empty role parameter")

        clean_tags = dict(tags) if tags else {}
        clean_tags.pop("role", None)

        # HTTP endpoint disabled - user management removed
        # Use CLI-only approach
        raise RuntimeError("add_entry only supports CLI mode - HTTP fallback disabled")

    def delete_memory(self, memory_id: str):
        """Delete a memory.  Best-effort; ignores 404s."""
        try:
            # Prefer CLI for symmetry with create_memory.
            self._run_cli(
                "delete-memory",
                "--memory-id",
                memory_id,
            )
            return
        except RuntimeError as cli_err:
            logging.info("[delete_memory] CLI failed, HTTP fallback disabled (no users): %s", cli_err)
            # HTTP fallback disabled - user management removed

    def put_context(self, memory_id: str, content: str):
        # Use new CLI subcommand.
        self._run_cli(
            "put-context",
            "--memory-id",
            memory_id,
            "--content",
            content,
        )

    # ------------------------------------------------------------------
    # Search
    # ------------------------------------------------------------------
    def search_memories(self, memory_id: str, query: str, top_k: int = 5) -> Dict[str, Any]:
        """Hybrid search within a memory. HTTP fallback disabled - use CLI only."""
        # HTTP endpoint disabled - user management removed
        raise RuntimeError("search_memories HTTP fallback disabled - use CLI-only client")

    # ------------------------------------------------------------------
    # Asset helper operations (static prompt assets)
    # ------------------------------------------------------------------
    def list_assets(self) -> List[str]:
        """Return list of available asset IDs via CLI; fallback to empty list."""
        try:
            out = self._run_cli("list-assets")
            out = out.strip()
            if not out:
                return []
            try:
                data = json.loads(out)
                assets = data.get("assets") or []
                if isinstance(assets, list):
                    return assets
            except json.JSONDecodeError:
                # Fallback: assume newline-separated output
                return [line.strip() for line in out.splitlines() if line.strip()]
        except Exception as cli_err:
            logger.debug("[list_assets] CLI failed: %s", cli_err)
        return []

    def get_asset(self, asset_id: str) -> str:
        """Return raw text content of a static asset via CLI; returns empty string on error."""
        try:
            out = self._run_cli("get-asset", "--id", asset_id)
            if out.strip():
                return out
            # If stdout empty, run raw subprocess to capture STDERR
            res = subprocess.run(
                ["mycelianCli", "--service-url", self.base_url, "get-asset", "--id", asset_id],
                capture_output=True,
                text=True,
            )
            if res.returncode == 0 and res.stderr.strip() and not res.stdout.strip():
                return res.stderr
            return res.stdout or res.stderr
        except Exception as cli_err:
            logger.debug("[get_asset] CLI failed: %s", cli_err)
            return ""

    # ------------------------------------------------------------------
    # Backwards-compatibility layer
    # ------------------------------------------------------------------
    def get_default_prompts(self) -> List[str]:  # pragma: no cover
        """Deprecated helper retained for backward compatibility.

        Historically returned the raw text bodies of default prompt assets.  It
        now proxies to ``list_assets`` so callers can migrate gradually.
        """
        logger.warning("get_default_prompts() is deprecated – use list_assets()/get_asset() instead")
        return self.list_assets()

    def await_consistency(self, memory_id: str) -> None:
        """Block until previous writes for the memory are consistent using CLI; fallback to short sleep."""
        try:
            self._run_cli("await-consistency", "--memory-id", memory_id)
            return
        except Exception as cli_err:
            logger.debug("[await_consistency] CLI failed: %s; falling back to sleep", cli_err)
            # Fallback: small delay similar to previous behaviour
            time.sleep(0.5)