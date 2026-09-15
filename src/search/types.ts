/** 搜索页共享类型（与 Go 侧 build/interfaces search 的 JSON 同形） */

import type { Post } from "../posts/types";

/** 插件搜索结果组（unit-6 §5.4 SearchProvider 贡献源） */
export interface PluginHitGroup {
    /** provider 名（= 插件名，如 "docs"） */
    provider: string;
    /** 命中项（标题/摘要/链接） */
    hits: { title: string; summary: string; url: string; updatedAt: string }[];
}

/** 搜索页的 initialState */
export interface SearchState {
    /** 当前生效的关键词（Go 侧已 trim） */
    q: string;
    /** scope：all（缺省）| blog | 插件 provider 名 */
    scope: string;
    /** 匹配结果（标题/正文 LIKE，创建时间倒序；blog 源） */
    results: Post[];
    /** 插件贡献结果（scope=blog 时空；旧前端忽略即无感） */
    pluginResults: PluginHitGroup[];
}
