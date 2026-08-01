/** 头像组件：avatarUrl 存在时显示图片，否则首字母方块（黑底白字 64px）；个人页/作者主页共用 */

import { v } from "../lib/theme";

export function LetterAvatar({ name, avatarUrl }: { name: string; avatarUrl?: string }) {
    if (avatarUrl) {
        return (
            <img
                src={avatarUrl}
                alt={`${name} 的头像`}
                style={{
                    width: 64,
                    height: 64,
                    borderRadius: 3,
                    objectFit: "cover",
                    border: `1px solid ${v.border}`,
                    flexShrink: 0,
                }}
            />
        );
    }
    return (
        <span
            style={{
                width: 64,
                height: 64,
                borderRadius: 3,
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
