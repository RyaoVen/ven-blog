/**
 * @file markdown 渲染单元测试（仓库根 src/lib/markdown.ts）
 * @description 自写渲染规则的 XSS 回归：admonition 自定义标题必须转义（issue #165）。
 * 由 CI 的 npm test（vitest run，frame/node）执行。
 */
import { describe, expect, it } from "vitest";
import { renderMarkdown } from "../../src/lib/markdown";

describe("renderMarkdown admonition 标题转义（issue #165）", () => {
    it("自定义标题含 HTML 时转义输出，无原始标签", () => {
        const { html } = renderMarkdown(":::warning <img src=x onerror=alert(1)>\n正文\n:::");
        expect(html).toContain("&lt;img src=x onerror=alert(1)&gt;");
        expect(html).not.toContain("<img src=x");
    });

    it("自定义标题含引号与 & 时正确转义", () => {
        const { html } = renderMarkdown(':::tip 他说的 "对" & 好\n内容\n:::');
        expect(html).toContain("他说的 &quot;对&quot; &amp; 好");
    });

    it("正常自定义标题不受影响", () => {
        const { html } = renderMarkdown(":::tip 小贴士\n内容\n:::");
        expect(html).toContain('<p class="ven-admonition-title">小贴士</p>');
    });

    it("默认标题不受影响", () => {
        const { html } = renderMarkdown(":::note\n内容\n:::");
        expect(html).toContain('<p class="ven-admonition-title">注意</p>');
    });
});
