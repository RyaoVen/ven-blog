/** 首字母方块头像：黑底白字（写法同文章详情页头像，放大到 64px）；个人页/作者主页共用 */

import { v } from "../lib/theme";

export function LetterAvatar({ name }: { name: string }) {
    return (
        <span
            style={{
                width: 64,
                height: 64,
                borderRadius: 2,
                background: v.text,
                display: "inline-flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 30,
                fontWeight: 700,
                fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
                color: v.bg,
                flexShrink: 0,
            }}
        >
            {name.slice(0, 1).toUpperCase()}
        </span>
    );
}
