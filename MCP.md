# MCP server for AI agents

`onigirazu mcp` speaks the [Model Context Protocol](https://modelcontextprotocol.io) over
stdio, so an AI agent — Claude Code, Claude Desktop, any MCP client — can look at the
inventory, plan and drift-check playbooks, run verify and compliance profiles and read the run
history. It is read-only unless started with `--allow-apply`.

```jsonc
// Claude Code: .mcp.json in the project, or Claude Desktop's config
{
  "mcpServers": {
    "onigirazu": {
      "command": "onigirazu",
      "args": ["mcp", "-i", "inventory/hosts.yml"]
    }
  }
}
```

```bash
claude mcp add onigirazu -- onigirazu mcp -i hosts.yml          # Claude Code
onigirazu mcp -i hosts.yml --allow-apply                        # also offers apply
```

## Tools

| Tool | Arguments | Returns |
|---|---|---|
| `inventory_list` | `inventory` | groups, hosts and host variables (JSON, passwords left out) |
| `inventory_host` | `host`, `inventory` | one host's variables (JSON) |
| `plan` | `playbook`, `inventory`, `limit`, `tags`, `extra_vars`, `become` | what apply would change, per host with diffs (Markdown); changes nothing |
| `drift` | the same | the drifting tasks per host (Markdown); changes nothing, recorded in the drift history |
| `verify` | the same | the playbook's `verify:` checks per host (JSON) |
| `comply` | `profile`, `inventory`, `limit`, `control_tags`, `min_severity` | the profile's controls per host with severities and fixes (Markdown) |
| `audit_runs` | `host`, `playbook`, `limit` | the last runs of the audit store (JSON) |
| `audit_run` | `id` | one run in full (JSON) |
| `module_doc` | `module` | a module's arguments and description |
| `apply` (only with `--allow-apply`) | as `plan` | runs the playbook — changes hosts |

`inventory` defaults to the server's `-i`. Credentials come from the environment and the
inventory as for the CLI (`BW_SESSION`, `VAULT_TOKEN`, ssh keys); the agent never sees them.
A failed tool call returns the error as its text with `isError: true`.

Logs and every message the tools would print go to stderr; stdout carries only the protocol
(JSON-RPC 2.0, one message per line, protocol version `2025-06-18`).

## Guidance for the agent

The `instructions` field of `initialize` tells the model that `plan` and `drift` are safe and
that `apply` exists only when allowed. A sensible workflow for an agent: `inventory_list` →
`plan` (and read the diffs) → ask the human → `apply` → `verify` or `comply`.
