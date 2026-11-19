# GoDaily MCP Server

GoDaily runs a read-only [Model Context Protocol](https://modelcontextprotocol.io)
(MCP) server. Connecting it to Claude, whether in claude.ai, Claude Desktop or
Claude Code, lets you ask questions in plain English and have Claude answer
from the real data:

- "What went out in yesterday's issue, and which links got clicked?"
- "Which sources drove the most clicks this month?"
- "Compare open rates for the last ten issues and find the best subject line."
- "What did we collect about iterators last week that didn't make the digest?"
- "How has subscriber growth looked week by week this quarter?"

Every tool is **read-only**. The server cannot edit drafts, send email, post to
social media or change subscribers, and it never returns subscriber email
addresses (only aggregate counts).

| | |
|---|---|
| **Endpoint** | `https://godaily.dev/api/mcp/` (note the trailing slash) |
| **Transport** | Streamable HTTP, stateless, JSON responses |
| **Auth** | `MCP_API_KEY` (read-only key) or `API_SECRET` |
| **Local transport** | `godaily mcp` (stdio) |
| **Code** | `pkg/mcpserver/`, mounted in `pkg/api/mux/mux.go` |

---

## 1. One-time setup: create the MCP key

The MCP endpoint accepts two credentials:

- **`MCP_API_KEY`**: a read-only key to hand out to yourself and colleagues.
  It only works on `/api/mcp/`; it is useless against the rest of the API
  (sending the digest, publishing social posts and so on).
- **`API_SECRET`**: the existing admin secret. It keeps working unchanged
  everywhere, including on `/api/mcp/`, so nothing needs rotating.

To create the MCP key:

1. Generate a random key:

   ```bash
   openssl rand -hex 32
   ```

2. In the Vercel project, open **Settings → Environment Variables** and add
   `MCP_API_KEY` with that value for the **Production** environment (and
   Preview if you want it there too).
3. Redeploy, since Vercel only applies new variables to new deployments.
4. Check it works:

   ```bash
   curl -s https://godaily.dev/api/mcp/ \
     -H "Authorization: Bearer $MCP_API_KEY" \
     -H "Content-Type: application/json" \
     -H "Accept: application/json, text/event-stream" \
     -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | jq '.result.tools[].name'
   ```

   You should see 16 tool names. A `401` means the key is wrong or the
   deployment predates the variable.

To revoke access, change `MCP_API_KEY` in Vercel and redeploy. Everyone using
the old key loses access at once; `API_SECRET` is unaffected.

---

## 2. Connect from claude.ai (web, desktop and mobile)

Custom connectors added on claude.ai sync to the Claude desktop and mobile
apps. claude.ai has two ways to pass the key. Use **request headers** if your
dialog offers them; otherwise use the **key in the URL**.

### Option A: request headers (preferred)

Anthropic is rolling out request-header authentication gradually. If you see a
**Request headers** section in the **Add custom connector** dialog, use this
option.

**Pro, Max or Free plans** (you add it for yourself):

1. Go to **Customize → Connectors**
   ([claude.ai/customize/connectors](https://claude.ai/customize/connectors)).
2. Select **Add**, then **Add custom connector**.
3. Enter:
   - **Name:** `GoDaily`
   - **Remote MCP server URL:** `https://godaily.dev/api/mcp/`

   Then click **Continue**.
4. Under **Authentication**, choose **No sign-in**. GoDaily doesn't use OAuth,
   so pick this even if another option is marked *Detected*.
5. Open **Request headers** (it may be under **Advanced**) and add a header:
   - **Header name:** `authorization`
   - **Value:** `Bearer <your MCP_API_KEY>`, including the word `Bearer` and
     a space, because claude.ai sends the value exactly as you type it.
   - Mark it **Required**.
6. Click **Add**.

**Team or Enterprise plans:** an Owner adds the connector once for the whole
organization.

1. Go to **Organization settings → Connectors**
   ([claude.ai/admin-settings/connectors](https://claude.ai/admin-settings/connectors)).
2. Select **Add**, then **Custom**. If asked for a connector type, choose
   **Web**.
3. Fill in the name, URL, **No sign-in** and the `authorization` request
   header exactly as in steps 3–5 above, then click **Add**.
4. Each member then goes to **Customize → Connectors**, finds **GoDaily** (it
   has a **Custom** label) and clicks **Connect**.

On a Team or Enterprise plan, the header is one shared credential for everyone
in the organization who connects, which suits a read-only key.

### Option B: key in the URL (when there is no Request headers section)

If your dialog has no **Request headers** section, put the key in the URL
instead:

1. Follow the same steps as Option A, but enter this as the
   **Remote MCP server URL**:

   ```
   https://godaily.dev/api/mcp/?key=<your MCP_API_KEY>
   ```

2. Choose **No sign-in** and skip request headers.

Things to know about this option:

- Only `MCP_API_KEY` works in the URL. `API_SECRET` is always refused there,
  so the admin secret can't end up in a URL by mistake.
- GoDaily removes the key from the URL before it reaches application logs.
  However, Vercel's own edge request logs and anyone who can see the connector
  configuration may still see the full URL. Treat the URL as a secret, and
  rotate `MCP_API_KEY` if it leaks. Switch to Option A once request headers
  are available to you.

### Using it in a chat

1. In a conversation, click **+**, choose **Connectors** and make sure
   **GoDaily** is turned on.
2. Ask a question, for example *"Using GoDaily, summarise the last week of
   engagement."*
3. The first time Claude calls a tool, it asks for permission. All GoDaily
   tools are read-only, so **Always allow** is safe. You can review or block
   individual tools under **Customize → Connectors → GoDaily → Tool
   permissions**.

You can't change authentication settings after adding a connector. To switch
keys or move from Option B to Option A, remove the connector and add it again.

---

## 3. Connect from Claude Code

Add the remote server, which works from any directory:

```bash
claude mcp add --transport http godaily https://godaily.dev/api/mcp/ \
  --header "Authorization: Bearer $GODAILY_MCP_KEY"
```

Add `--scope user` to make it available in every project. Check the
connection with `claude mcp list`, which should show
`godaily: https://godaily.dev/api/mcp/ (HTTP) - √ Connected`.

### Local development over stdio

To query a local or development database instead of production, run the
server over stdio from a checkout with a `.env`:

```bash
go build -o bin/godaily .
```

```bash
claude mcp add godaily-local -- sh -c "cd '$PWD' && exec ./bin/godaily mcp"
```

`godaily mcp` reads the same environment as every other `godaily` command
(`TURSO_URL` and so on, from `.env` in the working directory, hence the `cd`)
and writes logs only to stderr, keeping stdout clean for the protocol.

### Claude Desktop with a local database

Add this to `claude_desktop_config.json`, replacing `/path/to/godaily` with
the absolute path of your checkout:

```json
{
  "mcpServers": {
    "godaily-local": {
      "command": "/bin/sh",
      "args": ["-c", "cd /path/to/godaily && exec ./bin/godaily mcp"]
    }
  }
}
```

---

## Tools

| Tool | What it answers |
|---|---|
| `list_issues` | Issues newest first (no items), filterable by status, paginated. |
| `get_issue` | One issue with subject, summary, status and ordered items. |
| `get_issue_candidates` | Items from an issue's build window that **didn't** make it in. |
| `get_item` | One collected news item. |
| `search_items` | Search all collected items by text, source, tag, date and inclusion. |
| `list_item_facets` | Every source and tag with item counts, for filtering. |
| `metrics_summary` | Headline email engagement (deliveries, opens, clicks, rates). |
| `list_issue_engagement` | Per-issue engagement, sortable by any metric. |
| `get_issue_engagement` | One issue's stats plus its top 10 clicked links. |
| `top_items` | Most-clicked items across issues. |
| `top_tags` | Clicks by item tag. |
| `top_sources` | Clicks by item source. |
| `engagement_trend` | A metric over time, bucketed hourly, daily or weekly. |
| `issue_engagement_trend` | One issue's metric over time (defaults to 48h after send). |
| `subscriber_growth` | New, confirmed, unsubscribed and net subscribers over time. |
| `social_post_metrics` | Social posts with likes, reposts, comments and impressions. |

Common arguments:

- **Issue key:** a numeric ID, a date slug (`2026-11-18`), `latest` (the most
  recent *sent* issue) or `draft` (the most recent draft).
- **Window:** `period` (`day`, `week`, `month`, `year`, `all`) or explicit
  `from`/`to` dates (`YYYY-MM-DD`). Leave all of them out for all time. These
  rules are identical to the `/api/metrics/*` endpoints and share their code
  (`api.ParseDateWindow`).
- Rates such as `open_rate` and `click_rate` are fractions between 0 and 1.

Each tool advertises a JSON input schema (including the allowed values for
`status`, `sort`, `metric` and `bucket`) and an output schema. The server
rejects invalid arguments before any database query runs.

---

## How it works

```
claude.ai / Claude Code ──POST JSON-RPC──► Vercel /api/mcp/
                                              │
                    CORS → rate limit (exempt for valid keys) → logger
                                              │
                                     plugs.MCPAuth (MCP_API_KEY | API_SECRET)
                                              │
                         mcpserver.HTTPHandler (stateless, JSON responses)
                                              │
                              tools → domain repositories → Turso
```

- **Stateless:** every POST carries one complete JSON-RPC exchange and no
  session is kept between requests. Vercel can route consecutive requests to
  different instances, and the tools never need to call back into the client,
  so nothing is lost. `GET` and `DELETE` return `405`, as the spec requires for
  stateless servers.
- **JSON, not SSE:** responses are plain `application/json`, so nothing
  depends on long-lived streams surviving the serverless runtime.
- **Not rate limited when authenticated:** requests with a valid key skip the
  public per-IP limit. That matters because claude.ai calls from Anthropic's
  shared IP addresses, so a per-IP limit would throttle everyone together.
- **Fails closed:** if neither `MCP_API_KEY` nor `API_SECRET` is configured,
  the endpoint returns `503` instead of allowing anonymous access.
- **Credentials accepted:** `Authorization: Bearer <key>`, `X-API-Key: <key>`,
  or `?key=<MCP_API_KEY>`.

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `401 unauthorized` | Wrong key, or the deployment predates `MCP_API_KEY`; redeploy. In claude.ai, make sure the header value starts with `Bearer `. |
| claude.ai asks you to sign in | You chose **Sign in now** or **Sign in when needed**. Remove the connector and add it again with **No sign-in**. |
| claude.ai says the request header name isn't allowed | Use the `authorization` name from the list, not a custom header name. |
| `405` on GET | Expected. The server is stateless, so clients only POST. |
| `503 MCP server is not configured` | No keys are set in this environment. |
| Windowed metrics are all zero on a **local SQLite file** | A pre-existing local-only issue, unrelated to MCP (the REST API is affected the same way). The `file:` driver stores timestamps in a format SQLite's `datetime()` can't parse. Use `TURSO_URL=file:godaily.db?_time_format=sqlite` for new local databases. Turso in production is unaffected. |

## References

- [Add a connector that isn't in the directory](https://claude.com/docs/connectors/custom/remote-mcp) (claude.ai)
- [Connect Claude Code to tools via MCP](https://code.claude.com/docs/en/mcp)
- [MCP Streamable HTTP transport](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports)
