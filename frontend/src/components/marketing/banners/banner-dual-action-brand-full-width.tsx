"use client";

import type { ReactNode } from "react";
import { Flag05 } from "@untitledui/icons";
import { Button } from "@/components/base/buttons/button";
import { CloseButton } from "@/components/base/buttons/close-button";
import { FeaturedIcon } from "@/components/foundations/featured-icon/featured-icon";

export interface BannerAction {
    label: ReactNode;
    onClick?: () => void;
    loading?: boolean;
    disabled?: boolean;
}

export interface BannerDualActionBrandFullWidthProps {
    title: ReactNode;
    description?: ReactNode;
    /** Right-hand solid brand action (the primary CTA). */
    primaryAction: BannerAction;
    /** Left-hand light action. Rendered before the primary on desktop. */
    secondaryAction?: BannerAction;
    /** When provided, a close button appears top-right (mobile) / inline (md+). */
    onDismiss?: () => void;
    dismissLabel?: string;
    /** FeaturedIcon glyph; defaults to a flag. */
    icon?: ReactNode;
    className?: string;
}

/**
 * Untitled UI PRO `banner-dual-action-brand-full-width`, vendored and adapted:
 *
 * - TW4 → TW 3.4 class ports: `shadow-xs!` → `!shadow-xs`,
 *   `focus:outline-hidden` → `focus:outline-none` (in CloseButton), plus
 *   `focus-visible:outline` added so focus rings get an outline-style on TW3
 *   (TW3's `outline-2` sets width only).
 * - Copy/actions are props: the original had hardcoded cookie-banner text.
 * - `md:truncate` kept on the title but dropped from the description — app
 *   copy (month + headcount + instruction) is longer than cookie copy and
 *   must wrap rather than ellipsize.
 * - The secondary button gets `bg-white text-foreground`: UU's semantic
 *   `bg-primary` maps to this app's emerald (shadcn), which would render it
 *   identical to the primary button. On the dark strip, white is UU's intent.
 */
export const BannerDualActionBrandFullWidth = ({
    title,
    description,
    primaryAction,
    secondaryAction,
    onDismiss,
    dismissLabel = "Đóng",
    icon = Flag05,
    className,
}: BannerDualActionBrandFullWidthProps) => {
    return (
        <div
            className={`relative border-t border-brand_alt bg-brand-section_subtle md:border-t-0 md:border-b md:border-brand ${
                className ?? ""
            }`}
        >
            <div className="mx-auto flex max-w-container flex-col gap-4 p-4 md:flex-row md:items-center md:gap-3 md:px-8 md:py-3">
                <div className="flex flex-1 flex-col gap-4 md:w-0 md:flex-row md:items-center">
                    <FeaturedIcon
                        className="hidden md:flex"
                        icon={icon}
                        color="brand"
                        theme="dark"
                        size="lg"
                    />

                    <div className="flex flex-col gap-0.5 overflow-hidden lg:flex-row lg:gap-1.5">
                        <p className="pr-8 text-md font-semibold text-primary_on-brand md:truncate md:pr-0">
                            {title}
                        </p>
                        {description ? (
                            <p className="text-md text-tertiary_on-brand">{description}</p>
                        ) : null}
                    </div>
                </div>
                <div className="flex gap-2">
                    <div className="flex w-full flex-col-reverse gap-3 md:flex-row">
                        {secondaryAction ? (
                            <Button
                                color="secondary"
                                size="lg"
                                className="!shadow-xs ring-0 bg-white text-foreground hover:bg-white/90 hover:text-foreground"
                                isDisabled={secondaryAction.disabled}
                                isLoading={secondaryAction.loading}
                                onClick={secondaryAction.onClick}
                            >
                                {secondaryAction.label}
                            </Button>
                        ) : null}
                        <Button
                            color="primary"
                            size="lg"
                            isDisabled={primaryAction.disabled}
                            isLoading={primaryAction.loading}
                            onClick={primaryAction.onClick}
                        >
                            {primaryAction.label}
                        </Button>
                    </div>
                    {onDismiss ? (
                        <div className="absolute top-2 right-2 flex shrink-0 items-center justify-center md:static">
                            <CloseButton size="md" theme="dark" label={dismissLabel} onClick={onDismiss} />
                        </div>
                    ) : null}
                </div>
            </div>
        </div>
    );
};
