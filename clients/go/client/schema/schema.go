package schema

import _ "embed"

// toolsSchemaRaw holds the canonical JSON array describing every MCP tool.
//
//go:embed tools.schema.json
var toolsSchemaRaw []byte

// ToolsSchema returns the embedded tools schema JSON.
func ToolsSchema() []byte {
	return toolsSchemaRaw
}
