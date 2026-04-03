# xhs-cli

A CLI tool for Xiaohongshu (Little Red Book) — enabling AI Agents to directly search, read, publish, and interact with Xiaohongshu content.

Built on browser automation, no API reverse-engineering required.

> The core browser automation logic is ported from [xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp). This project restructures it from an MCP Server architecture into a lightweight CLI tool, removing the Docker and mcporter dependencies so that AI Agents can invoke it directly from the command line. Thanks to the original author for the open-source contribution.

[中文文档](../README.md)

## Installation

```bash
go install github.com/Suoyiran1/xhs-cli@latest
```

Or build from source:

```bash
git clone https://github.com/Suoyiran1/xhs-cli.git
cd xhs-cli
go build -o xhs .
```

## Quick Start

```bash
# 1. Login (QR code scan required on first use)
xhs login

# 2. Search notes
xhs search "travel tips" --json

# 3. View note details
xhs detail <note_id> --xsec-token <token> --json

# 4. Get homepage feed
xhs feed --json
```

## Commands

| Command | Description | Example |
|---------|-------------|---------|
| `xhs login` | QR code login | `xhs login` |
| `xhs login status` | Check login status | `xhs login status --json` |
| `xhs search` | Search notes | `xhs search "keyword" --sort 最新 --json` |
| `xhs detail` | Note detail + comments | `xhs detail <id> --xsec-token <t> --comments --json` |
| `xhs feed` | Homepage feed | `xhs feed --json` |
| `xhs user` | User profile | `xhs user <uid> --xsec-token <t> --json` |
| `xhs comment` | Post comment | `xhs comment <id> "content" --xsec-token <t>` |
| `xhs reply` | Reply to comment | `xhs reply <id> "content" --xsec-token <t> --comment-id <cid>` |
| `xhs like` | Like a note | `xhs like <id> --xsec-token <t>` |
| `xhs favorite` | Favorite a note | `xhs favorite <id> --xsec-token <t>` |

## Search Filters

```bash
xhs search "food" \
  --sort "最新" \          # 综合|最新|最多点赞|最多评论|最多收藏
  --type "视频" \          # 不限|视频|图文
  --time "一周内" \        # 不限|一天内|一周内|半年内
  --limit 10 \
  --json
```

> Note: Filter values use Chinese strings as they map directly to Xiaohongshu's UI elements.

## Global Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--headless` | Run browser in headless mode | `true` |
| `--bin` | Browser binary path | auto-detect |
| `--json` | Output JSON format | `false` |
| `-v, --verbose` | Verbose logging | `false` |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `ROD_BROWSER_BIN` | Chrome/Chromium binary path |
| `XHS_PROXY` | Proxy server (http/socks5) |
| `COOKIES_PATH` | Custom cookie file path |

## Cookie Storage

Cookies are stored at `~/.xhs-cli/cookies.json` by default. After logging in once, they are automatically loaded for subsequent sessions.

## Usage with AI Agents

Use with [Agent Reach](https://github.com/Panniantong/agent-reach) by adding to SKILL.md:

```markdown
## Xiaohongshu (xiaohongshu.com)
- Tool: xhs CLI
- Search: `xhs search "keyword" --limit 10 --json`
- Detail: `xhs detail <note_id> --xsec-token <token> --json`
- Like: `xhs like <note_id> --xsec-token <token>`
```

## Relationship with xiaohongshu-mcp

| | xiaohongshu-mcp | xhs-cli |
|---|---|---|
| Architecture | MCP Server + Docker | Single CLI binary |
| Dependencies | Docker + mcporter | Chrome/Chromium only |
| Invocation | MCP protocol | Command line + JSON stdout |
| Use case | MCP ecosystem integration | AI Agent CLI invocation |
| Core logic | Original | Ported from xiaohongshu-mcp |

## License

MIT
