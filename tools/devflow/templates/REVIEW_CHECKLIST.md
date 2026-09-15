# REVIEW 检查单（/devflow-review 渲染用）

> 输出到 `docs/reviews/REVIEW-{issue}.md`。结论分三级：**通过** / **有条件通过**（P1 限期改）/ **打回**（存在 P0）。
> 问题分级：P0 必须修复才能提 PR；P1 合并前修复；P2 可延后（记入 Issue）。

## A. 通用检查（所有 PR）

- [ ] **文档先行**：接口/表结构变更是否先更新了 docs/02、docs/03（对照 ADR 与契约）
- [ ] **架构边界**：模块间只通过 Service 调用，无跨模块直连 Mapper；Controller 无业务逻辑
- [ ] **事务**：`@Transactional` 位于 Service 公开方法；无自调用；事务内无远程调用
- [ ] **数据完整性**：金额 NUMERIC/BigDecimal；枚举走 VARCHAR+CHECK；流水只插入
- [ ] **错误处理**：使用 docs/03 §3 错误码表；无吞异常；无裸 printStackTrace
- [ ] **规范**：Conventional Commits 且带 `#issue`；分支命名符合 `{type}/{issue}-slug`
- [ ] **安全**：无密钥/令牌硬编码；新增外部输入有校验
- [ ] **测试**：核心路径有测试；命名与断言清晰

## B. 阶段专项（按 Issue 的 phase 标签选用）

### phase-1 基础物流
- [ ] 库存变更只经 InventorySupport；明细按 sku_id 排序加锁（死锁预防）
- [ ] before/change/after 流水一致；operator_type 正确（USER/AGENT/SYSTEM）
- [ ] 出库 confirm 校验 available；cancel 释放 locked

### phase-2 数据治理
- [ ] 指标 SQL 与 docs/05 §2 公式一致
- [ ] LLM 输出经过校验层（skuCode 存在性、confidence 范围）；prompt 版本号已落 run
- [ ] AI 建议落地走草稿单（人在环未被绕过）

### phase-3 MCP
- [ ] Tool 复用 Service、未直连 Mapper；description 完整
- [ ] 写操作 operator=agent:mcp；只创建草稿单

### phase-4 Webhook
- [ ] 事件监听用 AFTER_COMMIT；签名针对 raw body
- [ ] 出站先落库再投递；入站幂等键去重；时间戳窗口校验

## C. 评审报告骨架

```markdown
# REVIEW-{issue} · {标题}

> 分支: {branch} · 基线: main@{sha} · 日期: {date} · 评审人: agent 预审
> **结论**: {通过 | 有条件通过 | 打回}

## 问题清单

| # | 级别 | 位置 | 问题 | 建议 |
|---|---|---|---|---|
| 1 | P0 | server/.../Xxx.java:42 | {{…}} | {{…}} |

## 通过项摘要

{{A/B 检查单勾选情况与亮点}}

## 后续跟踪

- P1/P2 项已同步至 Issue #{issue}（由 /devflow-impl 修复后复审）
```
