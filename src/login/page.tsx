/** 登录页（直接访问/守卫跳转的落地页；导航栏登录走弹窗） */

import type { PageAppProps } from "../app/pageApp";
import { LoginForm } from "../lib/authForms";
import { Layout, useAuthEnabled } from "../lib/layout";
import { v } from "../lib/theme";

export default function LoginPage({ bootstrap }: PageAppProps) {
    const authEnabled = useAuthEnabled();
    return (
        <Layout>
            <div className="ven-card" style={{ maxWidth: 400, margin: "48px auto", padding: "32px 28px" }}>
                {authEnabled ? (
                    <>
                        <p className="ven-meta" style={{ margin: "0 0 6px" }}>
                            SIGN IN
                        </p>
                        <h1 style={{ fontSize: 22, marginBottom: 4 }}>登录</h1>
                        <p style={{ fontSize: 14, color: v.textSecondary, marginBottom: 24 }}>登录后可评论、点赞、收藏</p>
                        <LoginForm nextUrl={bootstrap.query.next} />
                        <p style={{ fontSize: 13, color: v.textSecondary, margin: "14px 0 0" }}>
                            没有账号？<a href="/register">去注册</a>
                        </p>
                    </>
                ) : (
                    <>
                        <p className="ven-meta" style={{ margin: "0 0 6px" }}>
                            SIGN IN
                        </p>
                        <h1 style={{ fontSize: 22, marginBottom: 4 }}>登录已关闭</h1>
                        <p style={{ fontSize: 14, color: v.textSecondary, marginBottom: 24 }}>
                            站长已关闭公开注册与登录入口，登录功能暂不可用；如你持有站长账号，请从后台
                            <a href="/admin/login">登入</a>。
                        </p>
                    </>
                )}
            </div>
        </Layout>
    );
}
