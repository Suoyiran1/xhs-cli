# xhs-cli

小红书 CLI 工具 — 让 AI Agent 直接操作小红书（搜索/阅读/发布/互动）。

基于浏览器自动化实现，无需逆向 API。核心逻辑移植自 [xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp)。

## 安装

```bash
go install github.com/Suoyiran1/xhs-cli@latest
```

或从源码编译：

```bash
git clone https://github.com/Suoyiran1/xhs-cli.git
cd xhs-cli
go build -o xhs .
```

## 快速开始

```bash
# 1. 登录（首次使用需扫码）
xhs login

# 2. 搜索笔记
xhs search "旅行攻略" --json

# 3. 查看笔记详情
xhs detail <note_id> --xsec-token <token> --json

# 4. 获取首页推荐
xhs feed --json
```

## 命令一览

| 命令 | 说明 | 示例 |
|------|------|------|
| `xhs login` | 扫码登录 | `xhs login` |
| `xhs login status` | 检查登录状态 | `xhs login status --json` |
| `xhs search` | 搜索笔记 | `xhs search "关键词" --sort 最新 --json` |
| `xhs detail` | 笔记详情+评论 | `xhs detail <id> --xsec-token <t> --comments --json` |
| `xhs feed` | 首页推荐 | `xhs feed --json` |
| `xhs user` | 用户主页 | `xhs user <uid> --xsec-token <t> --json` |
| `xhs comment` | 发表评论 | `xhs comment <id> "内容" --xsec-token <t>` |
| `xhs reply` | 回复评论 | `xhs reply <id> "内容" --xsec-token <t> --comment-id <cid>` |
| `xhs like` | 点赞 | `xhs like <id> --xsec-token <t>` |
| `xhs favorite` | 收藏 | `xhs favorite <id> --xsec-token <t>` |

## 搜索筛选

```bash
xhs search "美食" \
  --sort "最新" \          # 综合|最新|最多点赞|最多评论|最多收藏
  --type "视频" \          # 不限|视频|图文
  --time "一周内" \        # 不限|一天内|一周内|半年内
  --limit 10 \
  --json
```

## 全局参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--headless` | 无头模式 | `true` |
| `--bin` | 浏览器路径 | 自动检测 |
| `--json` | JSON 输出 | `false` |
| `-v, --verbose` | 详细日志 | `false` |

## 环境变量

| 变量 | 说明 |
|------|------|
| `ROD_BROWSER_BIN` | Chrome/Chromium 路径 |
| `XHS_PROXY` | 代理服务器 (http/socks5) |
| `COOKIES_PATH` | 自定义 Cookie 文件路径 |

## Cookie 存储

Cookie 默认保存在 `~/.xhs-cli/cookies.json`，登录一次后自动加载。

## 给 AI Agent 使用

配合 [Agent Reach](https://github.com/Panniantong/agent-reach) 使用，在 SKILL.md 中添加：

```markdown
## 小红书 (xiaohongshu.com)
- 工具: xhs CLI
- 搜索: `xhs search "关键词" --limit 10 --json`
- 详情: `xhs detail <note_id> --xsec-token <token> --json`
- 点赞: `xhs like <note_id> --xsec-token <token>`
```

## 致谢

核心浏览器自动化逻辑来自 [xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp)。

## License

MIT
