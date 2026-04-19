#!/bin/bash

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
MAGENTA='\033[1;35m'
NC='\033[0m'

echo -e "${GREEN}=== MCP SQL Server Installer ===${NC}"

REPO="TranHuyThang9999/mcp-sqlserver"
LATEST=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep '"tag_name"' | cut -d'"' -f4)
VERSION=${LATEST:-v1.1.4}

echo -e "${YELLOW}Version: $VERSION${NC}"

OS=$(uname -s)
ARCH=$(uname -m)
case "$OS-$ARCH" in
    Linux-x86_64) EXT="tar.gz"; DIR="mcp-sqlserver-linux-amd64" ;;
    Linux-arm64)   EXT="tar.gz"; DIR="mcp-sqlserver-linux-arm64" ;;
    Darwin-x86_64) EXT="tar.gz"; DIR="mcp-sqlserver-darwin-amd64" ;;
    Darwin-arm64)  EXT="tar.gz"; DIR="mcp-sqlserver-darwin-arm64" ;;
    *) echo "Unsupported: $OS-$ARCH"; exit 1 ;;
esac

URL="https://github.com/$REPO/releases/download/$VERSION/mcp-sqlserver-$DIR.$EXT"
TMP="/tmp/mcp-sqlserver.$EXT"

echo -e "${YELLOW}Downloading...${NC}"
curl -L -o "$TMP" "$URL"

echo -e "${YELLOW}Extracting...${NC}"
mkdir -p "$HOME/.mcp-sqlserver"
tar -xzf "$TMP" -C "$HOME/.mcp-sqlserver"
rm "$TMP"

EXE="$HOME/.mcp-sqlserver/$DIR/mcp-sqlserver"
chmod +x "$EXE"

echo -e "${GREEN}Installed: $EXE${NC}"

echo -e "${MAGENTA}>>> SQL SERVER CONFIGURATION${NC}"
read -p "Host     [localhost]: " HOST
HOST=${HOST:-localhost}
read -p "Port     [1433]: " PORT
PORT=${PORT:-1433}
read -p "User    [sa]: " USER
USER=${USER:-sa}
read -p "Password: " -s PASS
echo
read -p "Database [master]: " DB
DB=${DB:-master}

echo -e "${MAGENTA}>>> CONFIGURING CLIENTS${NC}"
echo "1. All clients"
echo "2. Codex only"
echo "3. Cursor/VS Code only"
read -p "Selection [1]: " CHOICE
CHOICE=${CHOICE:-1}

case "$CHOICE" in
    1|2)
        mkdir -p "$HOME/.codex"
        cat >> "$HOME/.codex/config.toml" << EOF

[mcp_servers.sqlserver]
command = "$EXE"
startup_timeout_sec = 10
tool_timeout_sec = 60

[mcp_servers.sqlserver.env]
SQL_SERVER_HOST = "$HOST"
SQL_SERVER_PORT = "$PORT"
SQL_SERVER_USER = "$USER"
SQL_SERVER_PASSWORD = "$PASS"
SQL_SERVER_DATABASE = "$DB"
SQL_SERVER_ENCRYPT = "disable"
SQL_SERVER_TRUST_CERT = "true"
MCP_SQLSERVER_MAX_ROWS = "500"
MCP_SQLSERVER_ALLOW_SCHEMA_CHANGES = "false"
MCP_SQLSERVER_ALLOW_DANGEROUS_SQL = "false"
MCP_SQLSERVER_ALLOW_PROCEDURE_CALLS = "true"
EOF
        echo -e "${GREEN}Codex: $HOME/.codex/config.toml${NC}"
        ;;
esac

case "$CHOICE" in
    1|3)
        mkdir -p "$HOME/.cursor"
        cat > "$HOME/.cursor/mcp.json" << EOF
{
  "mcpServers": {
    "sqlserver": {
      "command": "$EXE",
      "env": {
        "SQL_SERVER_HOST": "$HOST",
        "SQL_SERVER_PORT": "$PORT",
        "SQL_SERVER_USER": "$USER",
        "SQL_SERVER_PASSWORD": "$PASS",
        "SQL_SERVER_DATABASE": "$DB",
        "SQL_SERVER_ENCRYPT": "disable",
        "SQL_SERVER_TRUST_CERT": "true",
        "MCP_SQLSERVER_MAX_ROWS": "500",
        "MCP_SQLSERVER_ALLOW_SCHEMA_CHANGES": "false",
        "MCP_SQLSERVER_ALLOW_DANGEROUS_SQL": "false",
        "MCP_SQLSERVER_ALLOW_PROCEDURE_CALLS": "true"
      }
    }
  }
}
EOF
        echo -e "${GREEN}Cursor: $HOME/.cursor/mcp.json${NC}"
        ;;
esac

echo -e "${GREEN}=== DONE ===${NC}"
echo "Restart your client and use health_check tool."
