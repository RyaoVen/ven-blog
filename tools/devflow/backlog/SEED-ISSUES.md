# 首批工单种子（SEED ISSUES）

> 配合 `docs/10-milestones.md` 使用：M0 全部 + M1 首批。**不要用脚本批量创建**——每张工单都应经 `/devflow-issue` 受理（补齐验收标准 → 确认门 → 创建 → 挂里程碑），这是 Phase 0 的工作流练习本身。
> 【示范】= 带教者实现学员读码；【学员】= 学员独立实现带教者 review（对应 docs/08 双轨制）。

## M0 · 环境与基线就绪

| # | 标题 | 轨道 | 验收标准要点 |
|---|---|---|---|
| 1 | [phase-0][chore] 项目骨架：server + web + compose 三件套 | 【示范】 | docker 起 PG；后端 /actuator/health UP；前端首页可访问；目录符合 docs/01 §8 |
| 2 | [phase-0][chore] Flyway V1 基线迁移与种子数据 | 【示范】 | 启动即执行 V1__basic/V4__seed；psql \dt 与 docs/02 表清单一致 |
| 3 | [phase-0][docs] README 与 AGENTS.md | 【学员】 | README 含跑通步骤；AGENTS.md 固化 docs/09 工作流三原则与红线 |
| 4 | [phase-0][chore] VenDevFlow 首个闭环演练 | 【学员】 | 以"完善 README"为需求完整走 issue→ship；main 出现首个 squash 合入 |

## M1 · 基础物流可用（首批 10 项，受理顺序即建议实施顺序）

| # | 标题 | 轨道 | 验收标准要点 |
|---|---|---|---|
| 5 | [phase-1][feat] 供应商管理后端 CRUD | 【示范】 | 分页/详情/新增/修改/停启用；编码重复 409 SUPPLIER_CODE_DUPLICATED |
| 6 | [phase-1][feat] 供应商管理前端页面 | 【示范】 | 列表+搜索+弹窗表单；Axios 拦截器统一解包与错误提示 |
| 7 | [phase-1][feat] SKU 管理前后端 | 【学员】 | 含成本/售价/安全库存字段；lowStockOnly 过滤 |
| 8 | [phase-1][feat] 仓库管理前后端 | 【学员】 | 同构复用供应商页面的组件经验 |
| 9 | [phase-1][feat] 入库单主子表与状态机（不含库存联动） | 【示范】 | 草稿创建/详情/列表；confirm/cancel 迁移与 ORDER_STATUS_INVALID |
| 10 | [phase-1][feat] 出库单主子表与状态机 | 【学员】 | 与入库同构；customer_name 落库 |
| 11 | [phase-1][feat] 库存核心：InventorySupport 行锁与流水 | 【示范】 | 入库 complete 同事务加库存+写流水；明细按 sku_id 排序加锁 |
| 12 | [phase-1][feat] 出库锁定、扣减与取消回退 | 【学员】 | confirm 校验 available；ship 扣减；CONFIRMED 取消解锁 |
| 13 | [phase-1][feat] JWT 认证与操作人贯通 | 【示范】 | 登录/401/路由守卫；单据与流水显示操作人 |
| 14 | [phase-1][exercise] 并发超卖实验与乐观锁对比 | 【学员】 | 悲观锁下双端并发不超卖（截图）；乐观锁改造 + 压测对比小结 |

> 录入提示：标题照抄左列；受理时由 /devflow-issue 按 ISSUE_TEMPLATE 补全背景/范围/完整验收标准，并挂载对应里程碑（M0=#编号见 bootstrap 输出，M1 同理）。
