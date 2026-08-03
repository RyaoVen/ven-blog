/** 注册页（直接访问落地页；导航栏注册走弹窗） */

import type { PageAppProps } from "../app/pageApp";
import { RegisterForm } from "../lib/authForms";
import { Layout, useAuthEnabled } from "../lib/layout";
import { v } from "../lib/theme";

export default function RegisterPage({ bootstrap }: PageAppProps) {
    const authEnabled = useAuthEnabled();
    return (
        <Layout>
            <div className="ven-card" style={{ maxWidth: 400, margin: "48px auto", padding: "32px 28px" }}>
                {authEnabled ? (
                    <>
                        <p className="ven-meta" style={{ margin: "0 0 6px" }}>
                            SIGN UP
                        </p>
                        <h1 style={{ fontSize: 22, marginBottom: 4 }}>注册</h1>
                        <p style={{ fontSize: 14, color: v.textSecondary, marginBottom: 24 }}>注册即可评论、点赞、收藏</p>
                        <RegisterForm nextUrl={bootstrap.query.next} />
                        <p style={{ fontSize: 13, color: v.textSecondary, margin: "14px 0 0" }}>
                            已有账号？<a href="/login">去登录</a>
                        </p>
                    </>
                ) : (
                    <>
                        <p className="ven-meta" style={{ margin: "0 0 6px" }}>
                            SIGN UP
                        </p>
                        <h1 style={{ fontSize: 22, marginBottom: 4 }}>注册已关闭</h1>
                        <p style={{ fontSize: 14, color: v.textSecondary, marginBottom: 24 }}>
                            站长已关闭公开注册入口，暂不接受新账号注册。
                        </p>
                    </>
                )}
            </div>
        </Layout>
    );
}
