# ISSUE 模板（/devflow-issue 渲染用）

> 占位符 `{{...}}` 由 skill 填充；渲染后先给用户确认，再调用 gitcode.sh issue-create。

```markdown
## 背景

{{为什么做这件事：业务动机 / 缺陷现象（含复现步骤）/ 教学目标}}

## 目标与范围

{{做什么、明确不做什么（防止范围蔓延）}}

**类型**：{{feat | fix | refactor | docs | chore | exercise(教学练习)}}
**所属阶段**：{{phase-0 | phase-1 | phase-2 | phase-3 | phase-4 | infra}}
**关联设计文档**：{{docs/0x-xxx.md 的相关小节，如 docs/04-module-basic.md §3.1}}

## 验收标准

- [ ] {{可客观验证的标准 1，如：POST /api/v1/inbound-orders 创建草稿单返回 orderNo}}
- [ ] {{标准 2}}
- [ ] {{标准 3：如相关，注明"docs/0x 已同步更新"}}

## 补充信息

{{参考链接、数据、截图等，无则删除本节}}
```

**标签规范**：`phase-N`（所属教学阶段）+ 类型（`feat`/`fix`/`exercise`…），多个用英文逗号传给脚本。
