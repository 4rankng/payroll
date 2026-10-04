import { cx } from "@/utils/cx";

export interface AuthBrandMarkProps {
    className?: string;
}

/**
 * TingTing logo + wordmark lockup shared by the auth surfaces (W10 UU PRO
 * migration). Copy and asset are identical to the pre-migration pages.
 */
export const AuthBrandMark = ({ className }: AuthBrandMarkProps) => (
    <div className={cx("flex items-center gap-3", className)}>
        <img src="/logo-square.png" alt="TingTing logo" className="h-12 w-12 object-contain" />
        <p className="font-display text-[1.35rem] font-black leading-none tracking-[-0.03em] text-fg-primary">
            TingTing
        </p>
    </div>
);
