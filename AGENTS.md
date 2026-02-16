# Sub2API Project Notes

本仓库是 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 的 fork（origin: qyhfrank/sub2api）。

## 上游同步与发布流程

### Tag 命名规则

| 来源 | 格式 | 示例 |
|------|------|------|
| 上游 | `v{major}.{minor}.{patch}` | `v0.1.82` |
| 本地 | `v{major}.{minor}.{patch}.merge.N` | `v0.1.82.merge.0` |

本地永远不创建上游格式的 tag。序号从 0 开始，每次在同一上游版本上发布递增。

### CI Trigger 规则

| 事件 | CI | Security Scan | Release |
|------|:--:|:-------------:|:-------:|
| push branch（含 main） | Yes | Yes | No |
| push 上游 tag `v0.1.82` | No | No | No |
| push merge tag `v0.1.82.merge.1` | No | No | Yes |
| workflow_dispatch | - | - | Yes |

`release.yml` 只匹配 `v*.merge.*`，上游 tag 不会触发。Docker 镜像由 GoReleaser 构建推送到 GHCR，`latest` 始终等于最新 merge tag。

### 同步上游（sync-upstream）

当用户说"同步上游"、"sync upstream"或使用 `/sync-upstream` 时：

```
1. git fetch upstream --tags --force
2. 找到最新上游 tag：
   git tag -l 'v*' --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1
3. 检查是否已包含：
   git merge-base --is-ancestor <tag> HEAD
   - 已包含 → 告知用户已是最新
   - 未包含 → 继续下一步
4. Merge 上游 tag（不是 upstream/main 的最新 commit）：
   git merge <tag>
   ⚠️ 始终 merge 上游的 tag，不要 merge upstream/main 分支。
   上游 main 可能有未发布的不稳定改动，tag 才是正式发布版本。
   ⚠️ 仔细处理冲突，防止本地 features 退化。
   本仓库有大量自定义改动（Gemini 429 处理、MCP server、UI 改进等），
   merge 时必须确保这些不被上游覆盖。
   如果有不适合自动 merge 的内容，停下来通知用户。
5. 推送 commits 和上游 tags：
   git push origin main
   git push origin <tag>
   （分开推送，上游 tag 不触发任何 workflow，仅作版本基准参考）
6. 打 merge tag：
   - 查找已有序号：git tag -l '<tag>.merge.*' --sort=-v:refname | head -1
   - 无已有 → <tag>.merge.0，已有 N → <tag>.merge.{N+1}
   - git tag <新 merge tag>
7. 推送 merge tag（触发 Release workflow）：
   git push origin <新 merge tag>
8. 检查 CI 并等待完成：
   gh run list --workflow=backend-ci.yml --limit 1
   gh run list --workflow=release.yml --limit 1
```

### 发布本地修改（release）

本地有新改动但不需要同步上游时：

```
1. 确认改动已提交到 main
2. 找到当前上游版本的最新 merge tag，递增序号
   例：当前最新是 v0.1.82.merge.0 → 新 tag 为 v0.1.82.merge.1
3. git push origin main
4. git tag <新 merge tag>
5. git push origin <新 merge tag>
6. 检查 CI：gh run list --workflow=release.yml --limit 1
```

也可用 workflow_dispatch 重新构建已有 tag：

```
gh workflow run release.yml -f tag=v0.1.82.merge.0 -f simple_release=true
```

## MCP Tool Registry

MCP server（`mcp-server/sub2api_mcp/`）通过 Registry 模式管理工具注册，配合 `ALLOWED_TOOLS` 白名单限制数量。

Claude API 对请求 payload 有大小限制，安全上限约 **100 个工具**。当前配置 56 个，留有余量。启用新工具时编辑 `mcp-server/sub2api_mcp/__init__.py` 的 `ALLOWED_TOOLS` 集合，必要时取消对应模块的 import 注释，重启生效。Registry 启动时会验证白名单中每个名字都有对应的装饰函数，拼写错误会直接报错退出。

## CRS 同步已知限制

CRS 的 `GET /admin/sync/export-accounts` **不导出 Gemini 账户**。Sub2API 后端已实现 Gemini 解析逻辑（`crs_sync_service.go:744-961`），一旦 CRS 更新导出接口即可自动生效。

## 部署运维备忘

- **curl 不可用**：部署前置 openresty/WAF 会拒绝 curl 请求头，用 Python `urllib.request` 可正常通过。
- **Auth 方式**：admin token 以 `admin-` 开头时使用 `x-api-key` header，否则使用 `Authorization: Bearer` header（参见 `client.py:13-15`）。
