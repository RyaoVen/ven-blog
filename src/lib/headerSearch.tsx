/** 导航栏搜索框：回车或按钮跳转 /search?q=（SPA data-only 取数） */

import { FormEvent, useState } from "react";
import { navigate } from "../app/router";

export function HeaderSearch() {
    const [kw, setKw] = useState("");

    function onSubmit(event: FormEvent) {
        event.preventDefault();
        const q = kw.trim();
        if (q) {
            navigate(`/search?q=${encodeURIComponent(q)}`);
        }
    }

    return (
        <form className="ven-header-search" onSubmit={onSubmit} role="search">
            <input
                className="ven-input"
                type="search"
                placeholder="搜索文章…"
                value={kw}
                onChange={(e) => setKw(e.target.value)}
                aria-label="搜索文章"
            />
        </form>
    );
}
