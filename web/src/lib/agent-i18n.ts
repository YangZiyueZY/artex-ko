import { useCallback, useEffect, useMemo, useState } from "react";

import { useTranslations } from "next-intl";

import { api } from "@/lib/api";
import type { Agent } from "@/lib/types";

// 内置智能体（builtin）的名称·描述·提示词变量标签来自数据库种子数据，
// 且只有中文（db/db.go 的 builtinAgents、server/finding_retests.go 的
// 漏洞复测 智能体）。数据库值无法按语言分列存放，且种子使用 ON CONFLICT DO NOTHING，
// 对已安装的实例也不会生效，因此界面按 key 从消息文件
//（web/messages/{ko,en,zh,es,ja}.json 的 builtinAgents 命名空间）中查找译文显示，
// 找不到时原样使用数据库值。用户创建的智能体始终使用数据库值。

// retester 在数据库中以 builtin=false 播种（server FindingRetestAgentKey），
// 但它是系统内置的固定 key，因此也纳入翻译范围。builtin=true 的智能体（goals/planner/
// mainagent/worker/auto/pentest）会被 agent.builtin 过滤，无需列在这里。
const SYSTEM_SEEDED_AGENT_KEYS = new Set(["retester"]);

function isSystemSeeded(agent: { key: string; builtin?: boolean }): boolean {
  return !!agent.builtin || SYSTEM_SEEDED_AGENT_KEYS.has(agent.key);
}

// useAgentLabels 是返回内置智能体名称·描述·提示词变量标签（按当前语言）的
// 辅助函数集合。所有展示位置都通过这些辅助函数获取标签。
export function useAgentLabels() {
  const t = useTranslations("builtinAgents");

  // 调用方会把这些函数作为 useMemo·useCallback 的依赖（对话分组·筛选计算），
  // 因此只在语言（t）变化时创建新集合。若每次渲染都产生新对象，这些计算会反复执行。
  return useMemo(() => {
    // name 对内置智能体返回翻译名称，否则返回数据库名称（agent.name）。
    const name = (agent: { key: string; name: string; builtin?: boolean }): string => {
      if (isSystemSeeded(agent) && t.has(`${agent.key}.name`)) return t(`${agent.key}.name`);
      return agent.name;
    };

    // description 对内置智能体返回翻译描述，否则返回数据库描述（没有则为空字符串）。
    // 空描述的兜底文案仍由调用方按原样拼接。
    const description = (agent: { key: string; description?: string; builtin?: boolean }): string => {
      if (isSystemSeeded(agent) && t.has(`${agent.key}.description`)) return t(`${agent.key}.description`);
      return agent.description ?? "";
    };

    // nameByKey 是没有 Agent 对象、只有 key 时（对话分组等）使用的变体。
    // 它接收查询函数（返回 agent 的函数），查到则返回译文，查不到则返回 key。
    const nameByKey = (
      key: string,
      lookup: (key: string) => { key: string; name: string; builtin?: boolean } | undefined,
    ): string => {
      const agent = lookup(key);
      // biome-ignore lint/nursery/useNullishCoalescing: 名称为空（""）的智能体仍要像以前一样改用 key 显示，因此 || 是刻意为之。
      return (agent && name(agent)) || key;
    };

    // varLabel 翻译内置智能体的提示词变量标签（例如 goals 的“任务总目标”）。
    // 没有译文时原样使用数据库标签（fallback）。
    const varLabel = (agentKey: string, builtin: boolean, varName: string, fallback: string): string => {
      if ((builtin || SYSTEM_SEEDED_AGENT_KEYS.has(agentKey)) && t.has(`${agentKey}.vars.${varName}`)) {
        return t(`${agentKey}.vars.${varName}`);
      }
      return fallback;
    };

    return { name, description, nameByKey, varLabel };
  }, [t]);
}

// 智能体列表只被界面多处（审批记录等）用于 key → 显示名称的转换，
// 因此获取一次后在模块级共享。失败时在下次调用时重新获取。
let agentListPromise: Promise<Agent[]> | null = null;
function loadAgentList(): Promise<Agent[]> {
  if (!agentListPromise) {
    agentListPromise = api.agents().catch((err) => {
      agentListPromise = null;
      throw err;
    });
  }
  return agentListPromise;
}

// useAgentNameByKey 返回供只有 key 而没有智能体对象的界面（审批·拦截记录中的
// conv_agent_key 等）使用的转换器。内置智能体显示当前语言名称，
// 用户智能体显示数据库名称，不在列表中或尚未获取到时原样显示 key。
export function useAgentNameByKey(): (key: string) => string {
  const labels = useAgentLabels();
  const [agents, setAgents] = useState<Map<string, Agent>>(() => new Map());
  useEffect(() => {
    let alive = true;
    loadAgentList()
      .then((list) => {
        if (alive) setAgents(new Map(list.map((a) => [a.key, a])));
      })
      .catch(() => {
        // 获取不到列表时原样显示 key（下次挂载时重试）。
      });
    return () => {
      alive = false;
    };
  }, []);
  return useCallback((key: string) => labels.nameByKey(key, (k) => agents.get(k)), [labels, agents]);
}
