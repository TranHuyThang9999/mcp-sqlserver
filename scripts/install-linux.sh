#!/usr/bin/env bash
set -euo pipefail

server_name="${SERVER_NAME:-sqlserver}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root_dir="$(cd "$script_dir/.." && pwd)"
binary="$script_dir/mcp-sqlserver"

if [[ ! -f "$binary" ]]; then
  binary="$root_dir/mcp-sqlserver"
fi

if [[ ! -f "$binary" ]]; then
  echo "Cannot find $binary"
  echo "Download the Linux release archive that includes the mcp-sqlserver binary."
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required to update JSON/TOML config files on Ubuntu."
  echo "Install it with: sudo apt install python3"
  exit 1
fi

chmod +x "$binary"

read_default() {
  local prompt="$1"
  local default_value="$2"
  local value

  read -r -p "$prompt [$default_value]: " value
  if [[ -z "$value" ]]; then
    printf '%s' "$default_value"
  else
    printf '%s' "$value"
  fi
}

read_secret() {
  local prompt="$1"
  local value

  read -r -s -p "$prompt: " value
  echo >&2
  printf '%s' "$value"
}

echo
echo "MCP SQL Server installer"
echo "Binary: $binary"
echo

sql_host="$(read_default "SQL Server host" "localhost")"
sql_port="$(read_default "SQL Server port" "1433")"
sql_user="$(read_default "SQL Server user" "sa")"
sql_password="$(read_secret "SQL Server password")"
sql_database="master"
sql_encrypt="disable"
sql_trust_cert="true"
max_rows="500"
allow_schema="false"
allow_dangerous="false"
allow_procedures="true"

export MCP_SQLSERVER_INSTALL_NAME="$server_name"
export MCP_SQLSERVER_BINARY="$binary"
export SQL_HOST="$sql_host"
export SQL_PORT="$sql_port"
export SQL_USER="$sql_user"
export SQL_PASSWORD="$sql_password"
export SQL_DATABASE="$sql_database"
export SQL_ENCRYPT="$sql_encrypt"
export SQL_TRUST_CERT="$sql_trust_cert"
export MAX_ROWS="$max_rows"
export ALLOW_SCHEMA="$allow_schema"
export ALLOW_DANGEROUS="$allow_dangerous"
export ALLOW_PROCEDURES="$allow_procedures"
export MCP_SQLSERVER_ENV_JSON
MCP_SQLSERVER_ENV_JSON="$(python3 - <<'PY'
import json
import os

env = {
    "SQL_SERVER_HOST": os.environ["SQL_HOST"],
    "SQL_SERVER_PORT": os.environ["SQL_PORT"],
    "SQL_SERVER_USER": os.environ["SQL_USER"],
    "SQL_SERVER_PASSWORD": os.environ["SQL_PASSWORD"],
    "SQL_SERVER_DATABASE": os.environ["SQL_DATABASE"],
    "SQL_SERVER_ENCRYPT": os.environ["SQL_ENCRYPT"],
    "SQL_SERVER_TRUST_CERT": os.environ["SQL_TRUST_CERT"],
    "MCP_SQLSERVER_MAX_ROWS": os.environ["MAX_ROWS"],
    "MCP_SQLSERVER_ALLOW_SCHEMA_CHANGES": os.environ["ALLOW_SCHEMA"],
    "MCP_SQLSERVER_ALLOW_DANGEROUS_SQL": os.environ["ALLOW_DANGEROUS"],
    "MCP_SQLSERVER_ALLOW_PROCEDURE_CALLS": os.environ["ALLOW_PROCEDURES"],
}

print(json.dumps(env, separators=(",", ":")))
PY
)"

echo
echo "Choose clients to configure:"
echo "1. Codex"
echo "2. Gemini CLI"
echo "3. Claude Code"
echo "4. Cursor / VS Code style MCP JSON"
echo "5. Project .mcp.json"
echo "6. All"
selection="$(read_default "Selection" "6")"

write_codex() {
  local path="$HOME/.codex/config.toml"
  mkdir -p "$(dirname "$path")"
  CONFIG_PATH="$path" python3 - <<'PY'
import json
import os
import re

path = os.environ["CONFIG_PATH"]
name = os.environ["MCP_SQLSERVER_INSTALL_NAME"]
binary = os.environ["MCP_SQLSERVER_BINARY"]
env = json.loads(os.environ["MCP_SQLSERVER_ENV_JSON"])
begin = "# BEGIN mcp-sqlserver managed block"
end = "# END mcp-sqlserver managed block"

try:
    with open(path, "r", encoding="utf-8") as f:
        content = f.read()
except FileNotFoundError:
    content = ""

content = re.sub(r"\n?# BEGIN mcp-sqlserver managed block.*?# END mcp-sqlserver managed block\n?", "\n", content, flags=re.S)

def toml_string(value):
    return str(value).replace("\\", "\\\\").replace('"', '\\"')

lines = [
    begin,
    f"[mcp_servers.{name}]",
    f'command = "{toml_string(binary)}"',
    "startup_timeout_sec = 10",
    "tool_timeout_sec = 60",
    "",
    f"[mcp_servers.{name}.env]",
]
for key in sorted(env):
    lines.append(f'{key} = "{toml_string(env[key])}"')
lines.append(end)

new_content = content.rstrip() + "\n\n" + "\n".join(lines) + "\n"
with open(path, "w", encoding="utf-8") as f:
    f.write(new_content.lstrip())
PY
  echo " - Codex: $path"
}

write_json_mcp() {
  local path="$1"
  local trust="${2:-false}"
  mkdir -p "$(dirname "$path")"
  CONFIG_PATH="$path" TRUST_SERVER="$trust" python3 - <<'PY'
import json
import os

path = os.environ["CONFIG_PATH"]
name = os.environ["MCP_SQLSERVER_INSTALL_NAME"]
binary = os.environ["MCP_SQLSERVER_BINARY"]
env = json.loads(os.environ["MCP_SQLSERVER_ENV_JSON"])
trust = os.environ.get("TRUST_SERVER") == "true"

try:
    with open(path, "r", encoding="utf-8") as f:
        config = json.load(f)
except (FileNotFoundError, json.JSONDecodeError):
    config = {}

if not isinstance(config, dict):
    config = {}
config.setdefault("mcpServers", {})
server = {
    "command": binary,
    "env": env,
}
if trust:
    server["trust"] = True
    server["timeout"] = 30000
config["mcpServers"][name] = server

with open(path, "w", encoding="utf-8") as f:
    json.dump(config, f, indent=2)
    f.write("\n")
PY
  echo " - $path"
}

configure_claude_code() {
  local server_json
  server_json="$(python3 - <<'PY'
import json
import os

print(json.dumps({
    "type": "stdio",
    "command": os.environ["MCP_SQLSERVER_BINARY"],
    "env": json.loads(os.environ["MCP_SQLSERVER_ENV_JSON"]),
}, separators=(",", ":")))
PY
)"

  if command -v claude >/dev/null 2>&1; then
    claude mcp add-json "$server_name" "$server_json" --scope user
    echo " - Claude Code: user scope via claude mcp add-json"
  else
    local path="$HOME/.mcp-sqlserver/claude-code-add-json.json"
    mkdir -p "$(dirname "$path")"
    printf '%s\n' "$server_json" > "$path"
    echo " - Claude Code CLI not found. JSON saved at: $path"
    echo "   Later run: claude mcp add-json $server_name '$server_json' --scope user"
  fi
}

echo
echo "Configured:"
case "$selection" in
  1)
    write_codex
    ;;
  2)
    write_json_mcp "$HOME/.gemini/settings.json" true
    ;;
  3)
    configure_claude_code
    ;;
  4)
    write_json_mcp "$HOME/.cursor/mcp.json" false
    ;;
  5)
    write_json_mcp "$PWD/.mcp.json" false
    ;;
  6)
    write_codex
    write_json_mcp "$HOME/.gemini/settings.json" true
    configure_claude_code
    write_json_mcp "$HOME/.cursor/mcp.json" false
    write_json_mcp "$PWD/.mcp.json" false
    ;;
  *)
    echo "Unknown selection: $selection"
    exit 1
    ;;
esac

echo
echo "Restart the selected client app, then ask it to run the SQL Server MCP health_check tool."
