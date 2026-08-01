/** 后台独立登入页（不套博客导航/页脚）：全屏居中卡片 + LoginForm，成功跳 next 或 /admin */

import type { PageAppProps } from "../../app/pageApp";
import { LoginForm } from "../../lib/authForms";
import { GlobalStyle } from "../../lib/layout";
import { v } from "../../lib/theme";

const mono = "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace";

export default function AdminLoginPage({ bootstrap }: PageAppProps) {
    return (
        <div
            style={{
                minHeight: "100vh",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                padding: 24,
            }}
        >
            <GlobalStyle />
            <div className="ven-card" style={{ width: "100%", maxWidth: 380, padding: "36px 32px" }}>
                <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 24 }}>
                    <span
                        style={{
                            width: 28,
                            height: 28,
                            borderRadius: 3,
                            background: v.text,
                            display: "inline-flex",
                            alignItems: "center",
                            justifyContent: "center",
                            color: v.bg,
                            fontSize: 14,
                            fontWeight: 700,
                            fontFamily: mono,
                        }}
                    >
                        V
                    </span>
                    <div>
                        <div style={{ fontWeight: 700, fontSize: 16, lineHeight: 1.2 }}>ven-blog</div>
                        <span className="ven-meta" style={{ fontSize: 10 }}>
                            ADMIN PORTAL
                        </span>
                    </div>
                </div>
                <h1 className="ven-serif" style={{ fontSize: 22, marginBottom: 6 }}>
                    后台登入
                </h1>
                <p style={{ fontSize: 13.5, color: v.textSecondary, marginBottom: 24 }}>
                    仅站长可进入管理面板
                </p>
                <LoginForm nextUrl={bootstrap.query.next || "/admin"} />
                <p style={{ margin: "20px 0 0", fontSize: 13 }}>
                    <a href="/" style={{ color: v.textSecondary }}>
                        ← 返回博客
                    </a>
                </p>
            </div>
        </div>
    );
}
