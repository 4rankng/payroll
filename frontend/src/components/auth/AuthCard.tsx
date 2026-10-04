import type { ReactNode } from "react";

import { cx } from "@/utils/cx";

export interface AuthCardProps {
    children: ReactNode;
    /** Extra classes for the card surface (width overrides, layout tweaks). */
    className?: string;
    /** Padding override — defaults to the shared auth rhythm p-6 sm:p-8. */
    paddingClassName?: string;
}

/**
 * The centered auth card surface shared by OTP login, forgot password, and
 * the two reset screens (W10 UU PRO migration). Pages keep their own outer
 * wrapper (data attributes and safe-area padding stay page-owned).
 */
export const AuthCard = ({ children, className, paddingClassName = "p-6 sm:p-8" }: AuthCardProps) => (
    <div className={cx("w-full max-w-[470px] rounded-xl border border-input bg-card shadow-xs", className)}>
        <div className={paddingClassName}>{children}</div>
    </div>
);
