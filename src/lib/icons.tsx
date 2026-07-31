/** SVG 图标库：feather 风格描边图标（currentColor 继承文本色，深浅色自适应）。
 * filled 属性切换填充态（心/书签的激活态用），其余描边统一 1.8。 */

import { CSSProperties, ReactNode } from "react";

export interface IconProps {
    size?: number;
    filled?: boolean;
    className?: string;
    style?: CSSProperties;
}

function icon(props: IconProps, children: ReactNode) {
    const filled = props.filled ?? false;
    return (
        <svg
            width={props.size ?? 15}
            height={props.size ?? 15}
            viewBox="0 0 24 24"
            fill={filled ? "currentColor" : "none"}
            stroke="currentColor"
            strokeWidth={1.8}
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
            className={props.className}
            style={props.style}
        >
            {children}
        </svg>
    );
}

export const HeartIcon = (p: IconProps) =>
    icon(p, (
        <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
    ));

export const BookmarkIcon = (p: IconProps) => icon(p, <path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z" />);

export const SunIcon = (p: IconProps) =>
    icon(p, (
        <>
            <circle cx="12" cy="12" r="4" />
            <path d="M12 2v2 M12 20v2 M4.93 4.93l1.41 1.41 M17.66 17.66l1.41 1.41 M2 12h2 M20 12h2 M6.34 17.66l-1.41 1.41 M19.07 4.93l-1.41 1.41" />
        </>
    ));

export const MoonIcon = (p: IconProps) => icon(p, <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />);

export const LoginIcon = (p: IconProps) =>
    icon(p, <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4 M10 17l5-5-5-5 M15 12H3" />);

export const LogoutIcon = (p: IconProps) =>
    icon(p, <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4 M16 17l5-5-5-5 M21 12H9" />);

export const SearchIcon = (p: IconProps) =>
    icon(p, (
        <>
            <circle cx="11" cy="11" r="8" />
            <path d="M21 21l-4.35-4.35" />
        </>
    ));

export const PenIcon = (p: IconProps) =>
    icon(p, <path d="M12 20h9 M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z" />);

export const GridIcon = (p: IconProps) =>
    icon(p, (
        <>
            <rect x="3" y="3" width="7" height="7" />
            <rect x="14" y="3" width="7" height="7" />
            <rect x="14" y="14" width="7" height="7" />
            <rect x="3" y="14" width="7" height="7" />
        </>
    ));

export const TrashIcon = (p: IconProps) =>
    icon(p, <path d="M3 6h18 M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2 M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />);

export const XIcon = (p: IconProps) => icon(p, <path d="M18 6L6 18 M6 6l12 12" />);

export const MessageIcon = (p: IconProps) =>
    icon(p, <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />);

export const RssIcon = (p: IconProps) =>
    icon(p, (
        <>
            <path d="M4 11a9 9 0 0 1 9 9 M4 4a16 16 0 0 1 16 16" />
            <circle cx="5" cy="19" r="1" fill="currentColor" />
        </>
    ));

export const UserPlusIcon = (p: IconProps) =>
    icon(p, (
        <>
            <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
            <circle cx="8.5" cy="7" r="4" />
            <path d="M20 8v6 M23 11h-6" />
        </>
    ));

export const CheckIcon = (p: IconProps) => icon(p, <path d="M20 6L9 17l-5-5" />);
