import subprocess
import re
import uuid
import logging
import time
import os
import os.path

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
        self.mycelian_cli_binary = self._find_mycelian_cli_binary()

    def _find_mycelian_cli_binary(self) -> str:
        """Auto-detect mycelianCli binary, prefer local repo version."""
        # Try local repo binary first (from tools/benchmarker/ -> ../../bin/mycelianCli)
        script_dir = os.path.dirname(os.path.abspath(__file__))
        local_binary = os.path.join(script_dir, "..", "..", "bin", "mycelianCli")
        local_binary = os.path.normpath(local_binary)
        
        if os.path.isfile(local_binary) and os.access(local_binary, os.X_OK):
            logger.debug("Using local mycelianCli binary: %s", local_binary)
            return local_binary
        
        # Fall back to PATH
        logger.debug("Local mycelianCli binary not found, falling back to PATH")
        return "mycelianCli"

    # Regex patterns for parsing CLI output
    _VAULT_REGEX = re.compile(r"Vault created: ([a-f0-9\-]+)")
    _MEM_REGEX = re.compile(r"Memory created: ([a-f0-9\-]+)")



    def _run_cli(self, *args: str) -> str:
        """Run mycelianCli command and return stdout."""
        cmd = [self.mycelian_cli_binary, "--service-url", self.base_url] + list(args)
        logger.info("[CLI DEBUG] Full command: %s", cmd)
        logger.info("[CLI DEBUG] Command string: %s", " ".join(cmd))
        
        # Set working directory to client where prompt files are located
        script_dir = os.path.dirname(os.path.abspath(__file__))
        cli_working_dir = os.path.join(script_dir, "..", "..", "client")
        cli_working_dir = os.path.normpath(cli_working_dir)
        
        try:
            result = subprocess.run(
                cmd, 
                capture_output=True, 
                text=True, 
                timeout=30,
                check=True,
                cwd=cli_working_dir
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

    def _get_or_create_default_vault(self) -> str:
        """Get or create a default vault for entries when vault_id is not tracked."""
        if not hasattr(self, '_default_vault_id'):
            vault_title = self.generate_unique_vault_title("default")
            self._default_vault_id = self.create_vault(vault_title, "Default vault for benchmark entries")
            logger.info("Created default vault %s for entries", self._default_vault_id)
        return self._default_vault_id

    def create_vault(self, title: str, description: str = "") -> str:
        """Create vault via CLI."""
        # Validate title meets vault constraints
        if not self._is_valid_vault_title(title):
            raise ValueError(f"Invalid vault title '{title}': must be 1-50 chars, ASCII letters/digits/hyphens only")
        
        out = self._run_cli(
            "create-vault",
            "--title", title,
            "--description", description,
        )
        m = self._VAULT_REGEX.search(out)
        if not m:
            raise RuntimeError(f"Failed to parse vault ID from CLI output:\n{out}")
        return m.group(1)

    def list_vaults(self) -> List[Dict[str, str]]:
        """List vaults via CLI; returns list of {id,title} dicts.

        Parses the mycelianCli plain-text output where each line is:
        "<uuid>\t<title>" and a trailing "Total: N" line.
        """
        out = self._run_cli("list-vaults").strip()
        vaults: List[Dict[str, str]] = []
        if not out:
            return vaults
        for line in out.splitlines():
            line = line.strip()
            if not line or line.lower().startswith("total"):
                continue
            # Expect: "<uuid>    <title>"
            parts = line.split(None, 1)
            if len(parts) != 2:
                continue
            vaults.append({"id": parts[0], "title": parts[1]})
        return vaults

    def find_vault_by_title(self, title: str) -> Optional[str]:
        """Return vault_id for the first vault whose title matches exactly."""
        for v in self.list_vaults():
            if v.get("title") == title:
                return v.get("id")
        return None

    def get_or_create_vault(self, title: str, description: str = "") -> str:
        """Return existing vault_id by title or create a new one if absent."""
        existing = self.find_vault_by_title(title)
        if existing:
            logger.info("Using existing vault %s (%s)", existing, title)
            return existing
        return self.create_vault(title, description)

    def _is_valid_vault_title(self, title: str) -> bool:
        """Check if vault title meets constraints: 1-50 chars, ASCII letters/digits/hyphens only."""
        if not title or len(title) > 50:
            return False
        # Match the regex from server validation: ^[A-Za-z0-9\-]+$
        return re.match(r'^[A-Za-z0-9\-]+$', title) is not None

    def create_memory_in_vault(self, vault_id: str, title: str, memory_type: str = "NOTES", description: str = "") -> str:
        """Create memory in a specific vault via MCP tool call."""
        # This method should use the MCP client, but for now we'll use CLI as fallback
        # Store vault_id for future add_entry calls
        self._last_vault_id = vault_id
            
        out = self._run_cli(
            "create-memory",
            "--vault-id", vault_id,
            "--title", title,
            "--memory-type", memory_type,
            "--description", description,
        )
        m = self._MEM_REGEX.search(out)
        if not m:
            raise RuntimeError(f"Failed to parse memory ID from CLI output:\n{out}")
        return m.group(1)

    def create_memory(self, title: str, memory_type: str = "conversation", description: str = "", vault_id: str | None = None) -> str:
        """Create memory via CLI."""
        # If no vault provided, create a unique vault for this memory
        if vault_id is None:
            vault_title = self.generate_unique_vault_title("memory")
            vault_id = self.create_vault(vault_title, f"Auto-created vault for memory: {title}")
            logger.info("Created unique vault %s (%s) for memory %s", vault_id, vault_title, title)
        
        # Store vault_id for future add_entry calls
        self._last_vault_id = vault_id
            
        out = self._run_cli(
            "create-memory",
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
        # Need vault_id for get-context command
        vault_id = getattr(self, '_last_vault_id', None) or self._get_or_create_default_vault()
        out = self._run_cli(
            "get-context",
            "--vault-id", vault_id,
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
        # Need vault_id for list-entries command
        vault_id = getattr(self, '_last_vault_id', None) or self._get_or_create_default_vault()
        out = self._run_cli(
            "list-entries",
            "--vault-id", vault_id,
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
        # Need vault_id for put-context command
        vault_id = getattr(self, '_last_vault_id', None) or self._get_or_create_default_vault()
        self._run_cli(
            "put-context",
            "--vault-id", vault_id,
            "--memory-id", memory_id,
            "--content", context,
        )

    def add_entry(self, memory_id: str, raw_entry: str, summary: str, *, role: str, tags: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Add entry using CLI. Requires vault_id which we get by parsing memory_id or using a default."""
        # For CLI compatibility, we need vault_id. We'll use a default vault approach.
        # This is a limitation of CLI-only approach - we need to track vault_id per memory
        # For now, we'll try to get vault from memory context or use a default
        
        # Try to find vault_id from existing memory context
        vault_id = getattr(self, '_last_vault_id', None)
        if not vault_id:
            # Fallback: create or use a default vault
            vault_id = self._get_or_create_default_vault()
            
        out = self._run_cli(
            "create-entry",
            "--vault-id", vault_id,
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
            "--memory-id", memory_id,
            "--query", query,
            "--top-k", str(top_k),
        ).strip()

        try:
            return json.loads(out) if out else {"entries": []}
        except json.JSONDecodeError:
            logger.warning("Failed to parse search JSON from CLI output")
            return {"entries": []}

    # Deprecated asset helpers removed: get_asset, list_assets

    def await_consistency(self, memory_id: str) -> None:
        """Block until previous writes for the memory are consistent using CLI."""
        try:
            self._run_cli(
                "await-consistency",
                "--memory-id", memory_id,
            )
        except Exception:
            # Small delay as fallback
            time.sleep(0.5)

    def get_prompts(self, memory_type: str = "chat") -> Dict[str, Any]:
        """Fetch default prompt templates for a memory type via CLI.

        Returns a dict:
          {
            "version": str,
            "context_summary_rules": str,
            "templates": {
               "context_prompt": str,
               "entry_capture_prompt": str,
               "summary_prompt": str,
            }
          }
        """
        out = self._run_cli("get-prompts", "--memory-type", memory_type).strip()
        try:
            data = json.loads(out) if out else {}
        except json.JSONDecodeError as e:
            logger.error("Failed to parse get-prompts JSON: %s", e)
            raise RuntimeError(f"Invalid JSON from get-prompts: {e}")

        # Basic shape validation
        if not isinstance(data, dict):
            raise RuntimeError("get-prompts returned non-object payload")
        if "version" not in data or "context_summary_rules" not in data or "templates" not in data:
            raise RuntimeError("get-prompts payload missing required fields")
        if not isinstance(data["templates"], dict):
            raise RuntimeError("get-prompts 'templates' is not an object")
        return data

    def get_tools_schema(self) -> List[Dict[str, Any]]:
        """Fetch live MCP tools schema via CLI."""
        out = self._run_cli("get-tools-schema").strip()
        try:
            schema = json.loads(out)
            if isinstance(schema, list):
                return schema
            else:
                logger.warning("get-tools-schema returned non-list: %s", type(schema))
                return []
        except json.JSONDecodeError as e:
            logger.error("Failed to parse tools schema JSON: %s", e)
            raise RuntimeError(f"Invalid JSON from get-tools-schema: {e}")

