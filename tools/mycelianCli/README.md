# MycelianCli

Command line interface for managing Mycelian memory service.

## Usage

```bash
mycelianCli [command] [flags]
```

## Commands

- `create-vault` - Create a new vault
- `create-memory` - Create a new memory in a vault
- `create-entry` - Create a new entry for a memory
- `list-entries` - List entries for a memory
- `get-prompts` - Get default prompt templates
- `put-context` - Update context document for a memory
- `get-context` - Get context document for a memory

## Structured Logging

The CLI uses **zerolog** for structured JSON logging, following the project's ADR-0002 standard.

### Enable Debug Mode

```bash
mycelianCli --debug [command]
# or
mycelianCli -d [command]
```

### Log Levels

- **Info** (default): Shows basic operation status
- **Debug** (with `--debug`): Shows detailed structured logs with timing, parameters, and responses
- **Error**: Shows failures with context

### Debug Features

When debug mode is enabled, the CLI will output structured JSON logs showing:

1. **Request parameters**: vault_id, memory_id, service_url, content_len, etc.
2. **Timing information**: elapsed time for operations
3. **HTTP details**: method, URL, status codes, request/response dumps
4. **Operation results**: counts, status, verification attempts, content types

### Example Debug Output

#### Entry Operations
```bash
mycelianCli --debug create-entry --vault-id vault-123 --memory-id mem-456 --raw-entry "test" --summary "test summary"
```

#### Context Operations
```bash
mycelianCli --debug put-context --vault-id vault-123 --memory-id mem-456 --content "active context data"
mycelianCli --debug get-context --vault-id vault-123 --memory-id mem-456
```

#### Vault Operations
```bash
mycelianCli --debug create-vault --title "My Project Vault" --description "Project notes and context"
```

#### Memory Operations
```bash
mycelianCli --debug create-memory --vault-id vault-123 --title "My Project" --memory-type "PROJECT" --description "Project notes"
```

Will output structured JSON logs like:
```json
{"level":"debug","email":"user@example.com","display_name":"John Doe","time_zone":"UTC","service_url":"http://localhost:8080","time":"2025-07-27T13:48:16-07:00","message":"creating user"}
{"level":"debug","user_id":"user-123","email":"user@example.com","display_name":"John Doe","time_zone":"UTC","elapsed":0.045,"time":"2025-07-27T13:48:16-07:00","message":"create user completed"}
{"level":"debug","user_id":"user-123","title":"My Project","memory_type":"PROJECT","description":"Project notes","service_url":"http://localhost:8080","time":"2025-07-27T13:48:16-07:00","message":"creating memory"}
{"level":"debug","user_id":"user-123","memory_id":"mem-456","title":"My Project","memory_type":"PROJECT","elapsed":0.032,"time":"2025-07-27T13:48:16-07:00","message":"create memory completed"}
```

### Environment Variables

- `MYCELIAN_DEBUG=true` - Enables HTTP request/response logging in the SDK
- `DEBUG=true` - Alternative way to enable HTTP logging
- `LOG_LEVEL=debug` - Alternative way to set log level
- `MEMORY_SERVICE_URL` - Override the default service URL (default: http://localhost:8080)

### Log Output Format

Logs are output in structured JSON format to stderr, making them easy to parse and analyze:

- **Timestamp**: ISO-8601 format with timezone
- **Level**: debug, info, warn, error
- **Context**: Structured fields like user_id, memory_id, elapsed time
- **Message**: Human-readable operation description

## Troubleshooting

### Add Entry Failures

If `create-entry` fails after showing "Entry enqueued":

1. Use `--debug` flag to see detailed structured logs
2. Check the `elapsed` field to see timing
3. Look for error-level logs with failure details
4. Verify HTTP request/response in debug output
5. Check backend logs for processing errors

### Context Operations Issues

If `put-context` or `get-context` commands fail:

#### Put Context Failures
1. Use `--debug` to see content_len and timing
2. Check for "put context failed" error logs
3. Verify content is not empty and properly formatted
4. Look for HTTP request/response details
5. Check backend logs for processing errors

#### Get Context Issues
1. Use `--debug` to see response details
2. Check `type` field: "string", "json", "empty"
3. Look for "get context: not found" vs "get context failed"
4. Verify `content_len` and `total_keys` in response
5. For fragment requests, check `filtered_keys` vs `requested_fragments`

### User Operations Issues

If `create-user` commands fail:

#### Create User Failures
1. Use `--debug` to see request parameters and timing
2. Check for "create user failed" error logs
3. Verify email format is valid
4. Look for HTTP request/response details
5. Check backend logs for validation errors

### Memory Operations Issues

If `create-memory` commands fail:

#### Create Memory Failures
1. Use `--debug` to see request parameters and timing
2. Check for "create memory failed" error logs
3. Verify user_id exists and is valid
4. Ensure memory_type is supported (e.g., PROJECT, NOTES)
5. Look for HTTP request/response details
6. Check backend logs for validation errors

### No Entries Returned

If `list-entries` returns no results:

1. Use `--debug` to see structured request/response details
2. Check `count` and `entries_returned` fields in logs
3. Verify user/memory IDs in the request logs
4. Look for timing issues in `elapsed` fields
5. Check for any error-level logs indicating failures

### Log Analysis

Since logs are in JSON format, you can use tools like `jq` for analysis:

```bash
# Filter only error logs
mycelianCli --debug create-entry ... 2>&1 | jq 'select(.level=="error")'

# Extract timing information
mycelianCli --debug list-entries ... 2>&1 | jq '.elapsed'

# Show HTTP requests
mycelianCli --debug create-entry ... 2>&1 | jq 'select(.message=="HTTP request")'

# Context-specific analysis
mycelianCli --debug put-context ... 2>&1 | jq 'select(.message=="put context completed") | .elapsed'
mycelianCli --debug get-context ... 2>&1 | jq 'select(.message=="get context completed") | {type, content_len, total_keys}'

# User and Memory-specific analysis
mycelianCli --debug create-user ... 2>&1 | jq 'select(.message=="create user completed") | {user_id, email, elapsed}'
mycelianCli --debug create-memory ... 2>&1 | jq 'select(.message=="create memory completed") | {memory_id, title, memory_type, elapsed}'
```
