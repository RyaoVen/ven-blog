/** 时间格式化 */

/** 格式化 ISO 时间为 "YYYY-MM-DD HH:mm"。
 * 确定性输出，避免 SSR（Node  locale/时区）与客户端渲染结果不一致导致 hydration mismatch。 */
export function formatDateTime(iso: string): string {
    return iso.slice(0, 16).replace("T", " ");
}
