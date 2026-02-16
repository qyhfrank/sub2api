# CHANGES.md - Fork 自定义功能记录

本文件记录 fork 相对于上游 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 新增和修改的功能。

---

## Gemini 429 限流处理优化

- 429 响应细分处理：区分 overload、no-capacity、daily limit、parsed limit 等不同类型
- model 级别 cooldown：避免单个模型 429 影响其他模型
- no-capacity 场景：不触发 model cooldown，直接重试
- fast failover：daily limit 和 parsed 429 立即切换到下一个可用模型
- overload 重试：加固 reauth + project fallback 逻辑
- 对齐上游 1s × 60 的 no-capacity 重试策略
- model capacity 耗尽时 30s 重试 + failover
- 日志脱敏：记录非 overload 429 的上游消息（去除敏感信息）

## MCP Server

- 新增 Sub2API MCP server（`mcp-server/sub2api_mcp/`），支持 CLI 方式管理网关
- 新增 proxy management tools

## UI 改进

- model rate limit badges 改为垂直堆叠显示
- Gemini project_id auth URL 输入体验优化

## Antigravity

- 转发与测试支持 daily/prod 单 URL 切换
