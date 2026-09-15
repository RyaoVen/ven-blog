#!/usr/bin/env bash
# =============================================================================
# VenDevFlow · 里程碑 Bootstrap：按 docs/10-milestones.md 基线在 GitCode 创建 M0~M5
# 幂等：同名里程碑已存在则跳过。用法: bootstrap.sh [--dry-run]
# 前置: devflow.env 已配置（owner/repo/token）
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GC="$SCRIPT_DIR/gitcode.sh"
DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

# 里程碑基线：标题 | 截止时间(ISO8601) | 描述
# （与 docs/10-milestones.md §2/§3 保持同步，日期调整时两处一起改）
MILESTONES=(
"M0 环境与基线就绪|2026-09-18T18:00:00+08:00|三件套跑通、Flyway V1 基线、VenDevFlow 首个闭环 PR。出口准则见 docs/10-milestones.md §3-M0。"
"M1 基础物流可用|2026-10-09T18:00:00+08:00|主数据 CRUD、出入库单据状态机、库存行锁与流水、JWT 贯通。出口准则见 docs/10-milestones.md §3-M1。"
"M2 数据治理闭环|2026-10-23T18:00:00+08:00|指标看板、规则告警、AI 工作流与人在环建议采纳。出口准则见 docs/10-milestones.md §3-M2。"
"M3 智能体接入|2026-10-30T18:00:00+08:00|MCP Server 六工具、鉴权与 AGENT 审计、客户端联调。出口准则见 docs/10-milestones.md §3-M3。"
"M4 供应商集成|2026-11-06T18:00:00+08:00|出站签名推送与重试、入站验签幂等、物流状态回写。出口准则见 docs/10-milestones.md §3-M4。"
"M5 结业交付|2026-11-12T18:00:00+08:00|15 分钟全链路演示与结业材料。出口准则见 docs/10-milestones.md §3-M5。"
)

echo "== 现有里程碑 =="
existing="$("$GC" milestone-list open 2>/dev/null || true)"
echo "${existing:-（无）}"

echo ""
echo "== 计划 =="
for m in "${MILESTONES[@]}"; do
  IFS='|' read -r title due desc <<<"$m"
  if grep -qF "$title" <<<"${existing:-}"; then
    echo "跳过（已存在）: $title"
  elif $DRY_RUN; then
    echo "[dry-run] 将创建: $title  due=$due"
  else
    echo -n "创建: $title ... "
    "$GC" milestone-create "$title" "$desc" "$due" | jq -c '{number, due_on}' || echo "失败（检查报错后重跑，幂等可续）"
  fi
done

echo ""
echo "完成。工单录入见 tools/devflow/backlog/SEED-ISSUES.md（经 /devflow-issue 逐个受理并挂载里程碑）。"
