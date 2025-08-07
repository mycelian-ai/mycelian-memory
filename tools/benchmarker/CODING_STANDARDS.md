# Benchmarker Coding Standards

## Backend Interface Policy

### Single Source of Truth: CLI Only

**Rule**: All interaction with the Synapse backend MUST use the CLI interface. HTTP API calls are prohibited.

**Rationale**:
- **Consistency**: CLI commands are the authoritative interface used by developers and tests
- **Maintainability**: Single interface reduces complexity and potential for inconsistencies  
- **Reliability**: CLI commands handle authentication, error formatting, and edge cases consistently
- **Debugging**: CLI output is human-readable and easier to troubleshoot

### Implementation Guidelines

#### ✅ Correct Approach
```python
def create_memory(self, title: str, memory_type: str = "chat", description: str = "") -> str:
    """Create memory using CLI - single authoritative interface."""
    out = self._run_cli(
        "create-memory",
        "--user-id", self.user_id,
        "--vault-id", vault_id,
        "--title", title,
        "--memory-type", memory_type,
        "--description", description,
    )
    # Parse CLI output for memory ID
    return self._extract_id_from_output(out)
```

#### ❌ Prohibited Approach
```python
def create_memory(self, title: str) -> str:
    """DON'T DO THIS - fallback creates complexity."""
    try:
        # CLI attempt
        return self._create_via_cli(title)
    except Exception:
        # HTTP fallback - PROHIBITED
        return self._create_via_http(title)
```

### Error Handling

- **CLI failures**: Propagate CLI errors directly with context
- **No fallbacks**: Do not implement HTTP fallbacks or alternative interfaces
- **Clear messages**: Include CLI command and output in error messages

### Dependencies

- **Allowed**: `subprocess` for CLI execution
- **Prohibited**: `requests`, `httpx`, or other HTTP client libraries

### Testing

- All tests must use CLI interface through the same methods used in production
- Mock CLI responses, not HTTP responses
- Verify actual CLI commands in integration tests

### Migration Guidelines

When removing HTTP fallback code:
1. Remove HTTP client dependencies 
2. Remove fallback try/except blocks
3. Simplify methods to CLI-only implementation
4. Update error messages to reference CLI failures
5. Remove HTTP-specific imports and session management