import { useState } from "react";
import { User01 } from "@untitledui/icons";
import { cx } from "@/utils/cx";
import { type AvatarProps } from "./avatar";
import { AvatarOnlineIndicator, VerifiedTick } from "./base-components";

/**
 * Untitled UI PRO `avatar-profile-photo`, vendored and adapted:
 *
 * - TW4 → TW 3.4 spacing ports: TW4's dynamic fractional spacing values
 *   absent from the TW 3.4 scale (`size-18`, `p-0.75`, `p-1.25`,
 *   `p-1.75`) become arbitrary px equivalents.
 * - Token-collision ports: UU `bg-primary`/`bg-tertiary` (white surface /
 *   light-gray surface) collide with this app's shadcn names → `bg-white`
 *   / `bg-utility-gray-100`; UU `text-fg-quaternary` → `text-fg-quaternary`;
 *   `outline-avatar-contrast-border` → `outline-utility-gray-200` with
 *   the `outline` style class TW3 needs to paint rings.
 */
const styles = {
    sm: {
        root: "size-[4.5rem] p-[3px]",
        rootWithPlaceholder: "p-1",
        content: "",
        icon: "size-9",
        initials: "text-display-sm font-semibold",
        badge: "bottom-0.5 right-0.5",
    },
    md: {
        root: "size-24 p-1",
        rootWithPlaceholder: "p-[5px]",
        content: "shadow-xl",
        icon: "size-12",
        initials: "text-display-md font-semibold",
        badge: "bottom-1 right-1",
    },
    lg: {
        root: "size-40 p-1.5",
        rootWithPlaceholder: "p-[7px]",
        content: "shadow-2xl",
        icon: "size-20",
        initials: "text-display-xl font-semibold",
        badge: "bottom-2 right-2",
    },
};

const tickSizeMap = {
    sm: "2xl",
    md: "3xl",
    lg: "4xl",
} as const;

interface AvatarProfilePhotoProps extends AvatarProps {
    size: "sm" | "md" | "lg";
}

export const AvatarProfilePhoto = ({
    contrastBorder = true,
    size = "md",
    src,
    alt,
    initials,
    placeholder,
    placeholderIcon: PlaceholderIcon,
    verified,
    badge,
    status,
    className,
}: AvatarProfilePhotoProps) => {
    const [isFailed, setIsFailed] = useState(false);

    const renderMainContent = () => {
        if (src && !isFailed) {
            return (
                <img
                    src={src}
                    alt={alt}
                    onError={() => setIsFailed(true)}
                    className={cx(
                        "size-full rounded-full object-cover",
                        contrastBorder && "outline outline-1 -outline-offset-1 outline-utility-gray-200",
                        styles[size].content,
                    )}
                />
            );
        }

        if (initials) {
            return (
                <div className={cx("flex size-full items-center justify-center rounded-full bg-utility-gray-100 ring-1 ring-secondary_alt", styles[size].content)}>
                    <span className={cx("text-fg-quaternary", styles[size].initials)}>{initials}</span>
                </div>
            );
        }

        if (PlaceholderIcon) {
            return (
                <div className={cx("flex size-full items-center justify-center rounded-full bg-utility-gray-100 ring-1 ring-secondary_alt", styles[size].content)}>
                    <PlaceholderIcon className={cx("text-fg-quaternary", styles[size].icon)} />
                </div>
            );
        }

        return (
            <div className={cx("flex size-full items-center justify-center rounded-full bg-utility-gray-100 ring-1 ring-secondary_alt", styles[size].content)}>
                {placeholder || <User01 className={cx("text-fg-quaternary", styles[size].icon)} />}
            </div>
        );
    };

    const renderBadgeContent = () => {
        if (status) {
            return <AvatarOnlineIndicator status={status} size={tickSizeMap[size]} className={styles[size].badge} />;
        }

        if (verified) {
            return <VerifiedTick size={tickSizeMap[size]} className={cx("absolute", styles[size].badge)} />;
        }

        return badge;
    };

    return (
        <div
            className={cx(
                "relative flex shrink-0 items-center justify-center rounded-full bg-white ring-1 ring-secondary_alt",
                styles[size].root,
                (!src || isFailed) && styles[size].rootWithPlaceholder,
                className,
            )}
        >
            {renderMainContent()}
            {renderBadgeContent()}
        </div>
    );
};
