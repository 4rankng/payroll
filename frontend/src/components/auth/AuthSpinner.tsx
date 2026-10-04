import type { SVGProps } from "react";

import { cx } from "@/utils/cx";

export interface AuthSpinnerProps extends Omit<SVGProps<SVGSVGElement>, "children"> {
    className?: string;
}

/**
 * Loading spinner for the auth surfaces, mirroring the vendored UU button's
 * inline-SVG loading icon (background circle + spinning dash). Pure SVG —
 * no icon-package import, no extra dependency.
 */
export const AuthSpinner = ({ className, ...rest }: AuthSpinnerProps) => (
    <svg fill="none" data-icon="loading" viewBox="0 0 20 20" className={cx("size-5 shrink-0", className)} {...rest}>
        {/* Background circle */}
        <circle className="stroke-current opacity-30" cx="10" cy="10" r="8" fill="none" strokeWidth="2" />
        {/* Spinning circle */}
        <circle
            className="origin-center animate-spin stroke-current motion-reduce:animate-none"
            cx="10"
            cy="10"
            r="8"
            fill="none"
            strokeWidth="2"
            strokeDasharray="12.5 50"
            strokeLinecap="round"
        />
    </svg>
);
