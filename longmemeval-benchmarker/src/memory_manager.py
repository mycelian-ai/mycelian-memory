"""Memory management utilities for the runner."""

from typing import Any, Dict, Optional
import asyncio
from src.async_utils import run as run_async
import uuid


class MemoryManager:
    """Handles vault and memory creation/management."""

    def __init__(self, mcp_client: Any, debug: bool = False):
        self._mcp = mcp_client
        self._debug = debug

    def _log(self, msg: str) -> None:
        if self._debug:
            print(msg)

    def _call_tool(self, name: str, arguments: Dict[str, Any]) -> Any:
        """Call an MCP tool by name."""
        # Find tool by name
        async def _get_tools():
            return await self._mcp.get_tools()  # type: ignore[attr-defined]

        tools = run_async(_get_tools())
        tool = None
        for t in tools:
            if getattr(t, "name", None) == name:
                tool = t
                break

        if tool is None:
            raise RuntimeError(f"MCP tool not found: {name}")

        if self._debug:
            self._log(f"[memory_manager] calling {name} with {arguments}")

        # Invoke tool
        if hasattr(tool, "ainvoke"):
            result = run_async(tool.ainvoke(arguments))  # type: ignore[attr-defined]
        elif hasattr(tool, "invoke"):
            result = tool.invoke(arguments)  # type: ignore[attr-defined]
        else:
            raise RuntimeError(f"Tool {name} has no invoke method")

        return result

    def ensure_vault(self, title: Optional[str], vault_id: Optional[str]) -> str:
        """Ensure a vault exists, creating if necessary."""
        if vault_id:
            return vault_id
        if not title:
            raise ValueError("either vault_id or vault_title is required")

        # Check if vault exists
        out = self._call_tool("list_vaults", {})
        vaults_iter = []

        # Parse the response - it could be a JSON string or already parsed
        if isinstance(out, str):
            try:
                import json
                parsed = json.loads(out)
                if isinstance(parsed, list):
                    vaults_iter = parsed
                elif isinstance(parsed, dict):
                    vaults_iter = parsed.get("vaults", []) or parsed.get("items", [])
            except:
                pass
        elif isinstance(out, list):
            vaults_iter = out
        elif isinstance(out, dict):
            vaults_iter = out.get("vaults", []) or out.get("items", [])

        if self._debug:
            self._log(f"[memory_manager] found {len(vaults_iter)} existing vaults")

        target = (title or "").strip().lower()
        for v in vaults_iter:
            if not isinstance(v, dict):
                continue
            vt = v.get("title", "")
            if self._debug and vt:
                self._log(f"[memory_manager] checking vault: '{vt}' vs target: '{target}'")
            if isinstance(vt, str) and vt.strip().lower() == target:
                vid = v.get("id") or v.get("vault_id") or v.get("vaultId")
                if vid:
                    self._log(f"[memory_manager] found existing vault: {vid}")
                    return vid

        # Create new vault
        try:
            created = self._call_tool("create_vault", {"title": title, "description": "longmemeval run"})
        except Exception as e:
            # Handle duplicate vault - check both the exception message and any nested error
            msg = str(e)
            # The error might be wrapped in a ToolException, so check comprehensively
            is_duplicate = (
                "duplicate" in msg.lower() or
                "23505" in msg or
                "already exists" in msg.lower() or
                "vaults_actor_id_title_key" in msg
            )
            if is_duplicate:
                self._log(f"[memory_manager] vault '{title}' already exists, searching for it...")
                # Re-list to find the vault
                out2 = self._call_tool("list_vaults", {})
                vaults_iter2 = []

                # Parse the response
                if isinstance(out2, str):
                    try:
                        import json
                        parsed = json.loads(out2)
                        if isinstance(parsed, list):
                            vaults_iter2 = parsed
                        elif isinstance(parsed, dict):
                            vaults_iter2 = parsed.get("vaults", []) or parsed.get("items", [])
                    except:
                        pass
                elif isinstance(out2, list):
                    vaults_iter2 = out2
                elif isinstance(out2, dict):
                    vaults_iter2 = out2.get("vaults", []) or out2.get("items", [])

                for v in vaults_iter2:
                    if not isinstance(v, dict):
                        continue
                    vt = v.get("title", "")
                    if isinstance(vt, str) and vt.strip().lower() == target:
                        vid2 = v.get("id") or v.get("vault_id") or v.get("vaultId")
                        if vid2:
                            self._log(f"[memory_manager] found vault after duplicate error: {vid2}")
                            return vid2

                # If we still can't find it, it might be a race condition
                # or the title comparison is failing. Log more details.
                self._log(f"[memory_manager] ERROR: Could not find vault '{title}' after duplicate error")
                self._log(f"[memory_manager] Available vaults: {[v.get('title') for v in vaults_iter2 if isinstance(v, dict)]}")
            raise

        # Extract vault ID from response - handle different response formats
        vid = None
        if isinstance(created, dict):
            # Try all possible field names for vault ID
            vid = created.get("vaultId") or created.get("id") or created.get("vault_id") or created.get("vaultID")
            # If it's the whole vault object, we need just the ID
            if not vid and "vaultId" in str(created):
                # Parse if it's a JSON string
                try:
                    import json
                    parsed = json.loads(str(created)) if isinstance(created, str) else created
                    vid = parsed.get("vaultId") or parsed.get("id")
                except:
                    pass
        elif isinstance(created, str):
            # Could be just the ID or a JSON string
            if created.startswith('{'):
                try:
                    import json
                    parsed = json.loads(created)
                    vid = parsed.get("vaultId") or parsed.get("id")
                except:
                    vid = created
            else:
                vid = created

        if not vid:
            self._log(f"[memory_manager] ERROR: create_vault response: {created}")
            raise RuntimeError(f"create_vault did not return an id, got: {type(created).__name__}")

        self._log(f"[memory_manager] created new vault '{title}': {vid}")
        return vid

    def ensure_memory(self, vault_id: str, title: str, memory_type: str = "NOTES") -> str:
        """Ensure a fresh memory exists; never reuse an existing same-title memory.

        If a duplicate-title error is encountered, generate a unique title by
        appending a deterministic numeric suffix, and as a last resort a UUID
        suffix, then retry creation. This prevents cross-run contamination.
        """

        def _try_create(with_title: str) -> Optional[str]:
            try:
                created_local = self._call_tool(
                    "create_memory_in_vault",
                    {"vault_id": vault_id, "title": with_title, "memory_type": memory_type, "description": ""},
                )
            except Exception as exc:
                # Detect duplicate title; signal to caller via None with flag
                msg = str(exc).lower()
                is_duplicate = (
                    "duplicate" in msg or
                    "already exists" in msg or
                    "23505" in msg or
                    "unique" in msg or
                    "memories_actor_id_title_key" in msg
                )
                if is_duplicate:
                    return None  # signal duplicate
                # Non-duplicate error: re-raise
                raise

            # Extract memory ID from response
            mid_local: Optional[str] = None
            if created_local:
                if isinstance(created_local, str):
                    if created_local.startswith('{'):
                        try:
                            import json
                            parsed = json.loads(created_local)
                            mid_local = parsed.get("memoryId") or parsed.get("id") or parsed.get("memory_id")
                        except Exception:
                            mid_local = created_local
                    else:
                        mid_local = created_local
                elif isinstance(created_local, dict):
                    mid_local = created_local.get("memoryId") or created_local.get("id") or created_local.get("memory_id")
            return mid_local

        # First attempt with the provided title
        mid = _try_create(title)
        if mid:
            self._log(f"[memory_manager] created memory: {mid}")
            return mid

        # On duplicate, try a small set of numeric suffixes deterministically
        for i in range(2, 6):  # __2 .. __5
            candidate = f"{title}__{i}"
            mid = _try_create(candidate)
            if mid:
                self._log(f"[memory_manager] created memory with suffix: {candidate} -> {mid}")
                return mid

        # Last resort: UUID suffix to guarantee uniqueness
        unique_suffix = uuid.uuid4().hex[:8]
        candidate = f"{title}__{unique_suffix}"
        mid = _try_create(candidate)
        if mid:
            self._log(f"[memory_manager] created memory with uuid suffix: {candidate} -> {mid}")
            return mid

        # If we reach here, something else went wrong (non-duplicate errors re-raised earlier)
        raise RuntimeError(f"Failed to create a new memory in vault {vault_id} after duplicate-title retries")

    def search_memories(self, memory_id: str, query: str, top_ke: int = 5, top_kc: int = 2) -> Dict[str, Any]:
        """Search memories and normalize the response.

        Args:
            memory_id: The UUID of the memory
            query: Search query text
            top_ke: Number of entry results (0-10, default 5)
            top_kc: Number of context shard results (1-3, default 2)
        """

        args: Dict[str, Any] = {
            "memory_id": memory_id,
            "query": query,
            "top_ke": int(top_ke),
            "top_kc": int(top_kc)
        }
        out = self._call_tool("search_memories", args)

        # Parse response if it's a JSON string
        if isinstance(out, str):
            try:
                import json
                return json.loads(out)
            except Exception:
                if self._debug:
                    self._log(f"[memory_manager] Could not parse search response: {out[:100]}")
                return {}
        elif isinstance(out, dict):
            return out
        else:
            return {}
