/** 搜索页共享类型（与 Go 侧 build/interfaces search 的 JSON 同形） */

import type { Post } from "../posts/types";

/** 搜索页的 initialState */
export interface SearchState {
    /** 当前生效的关键词（Go 侧已 trim） */
    q: string;
    /** 匹配结果（标题/正文 LIKE，创建时间倒序） */
    results: Post[];
}
