# pkg/mcpserver — MCP tool guide

This package exposes GoDaily to AI agents over the
[Model Context Protocol](https://modelcontextprotocol.io) using the official Go
SDK (`github.com/modelcontextprotocol/go-sdk`). For setup and connection
instructions (claude.ai, Claude Code, Claude Desktop), see
[`docs/features/mcp-server.md`](../../docs/features/mcp-server.md).

## Layout

| File | Purpose |
|---|---|
| `server.go` | `New`, `Deps`, server instructions, shared helpers (`readOnly`, `inputSchema`) |
| `http.go` | `HTTPHandler`: stateless Streamable HTTP, mounted at `/api/mcp` in `pkg/api/mux` |
| `params.go` | `Window` (period/from/to), limit and pagination helpers |
| `issues.go`, `items.go`, `metrics.go` | One file per tool group |
| `*_test.go` | In-memory client and gomock tests (see `helpers_test.go`) |

Auth lives in `pkg/api/plugs/mcp_auth.go`, not here, so the package works
unchanged over stdio (`godaily mcp`, see `pkg/cmd/mcp.go`).

## Rules

1. **Read-only.** Every tool uses `readOnly(title)` annotations, and
   `TestServer_ListTools` fails if one doesn't. Adding a tool that writes
   anything (drafts, email, social, subscribers) needs a separate design
   discussion first: it would need its own key, `DestructiveHint`, and a
   confirmation story.
2. **Depend on narrow interfaces.** Add any new repository to `Deps` and
   `DepsFromApp`, never the whole `App`, mirroring `pkg/api/handlers`.
3. **No personal data.** Never return subscriber email addresses or tokens;
   expose aggregates only.
4. **Match the HTTP API.** When a tool mirrors an `/api/...` endpoint, reuse its
   validation values and defaults (for example `api.ParseDateWindow`) so both
   surfaces behave identically.
5. **Mask internal errors.** Return user-facing errors for bad input or missing
   records. Wrap database failures with `internalError`, which logs the cause
   and returns a generic message.

## Adding a tool

```go
type WidgetInput struct {
	Window                                             // adds period/from/to
	Kind  string `json:"kind,omitempty"  jsonschema:"What kind of widget."`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum rows, at most 100. Defaults to 10."`
}

type WidgetOutput struct {
	Widgets []domain.Widget `json:"widgets"` // outputs must be objects: wrap slices
}

mcp.AddTool(s, &mcp.Tool{
	Name:        "list_widgets",
	Description: "What it returns and when to use it. The model reads this.",
	InputSchema: inputSchema[WidgetInput](withEnums(windowEnums, map[string][]string{
		"kind": {"small", "large"},
	})),
	Annotations: readOnly("List widgets"),
}, t.listWidgets)
```

Schema conventions (inferred by `github.com/google/jsonschema-go`):

- Fields **without** `omitempty`/`omitzero` become **required**.
- The `jsonschema:"..."` tag is the property description. Every property needs
  one (`TestServer_InputSchemas` checks this).
- Use `inputSchema[T](enums)` to advertise allowed values. The SDK then
  rejects invalid input before your handler runs.
- Unknown properties are rejected, so typos surface as errors.

Then:

1. Add the tool name to `wantTools` in `server_test.go`.
2. Write a test with `setup(t)` plus `decode[Out]` / `toolError`. These go
   through the real protocol, including input and output schema validation.
3. Run `go test ./pkg/mcpserver/ ./pkg/api/mux/`.
4. Update the tool table in `docs/features/mcp-server.md`.
5. If the tool changes in a way clients should notice, bump `Version`.
