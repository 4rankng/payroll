import { cx } from "@/utils/cx";

const sizes = {
    xs: "size-2",
    sm: "size-3",
    md: "size-3.5",
    lg: "size-4",
    xl: "size-4.5",
    "2xl": "size-5 ring-[1.67px]",
};

interface AvatarCompanyIconProps {
    size: "xs" | "sm" | "md" | "lg" | "xl" | "2xl";
    src: string;
    alt?: string;
}

export const AvatarCompanyIcon = ({ size, src, alt }: AvatarCompanyIconProps) => (
    <img
        src={src}
        alt={alt}
        // Token ports (not in the W1 bridge): `bg-primary-25` and UU's
        // `ring-bg-primary` page-bg ring → white; light-only app.
        className={cx("bg-white absolute -right-0.5 -bottom-0.5 rounded-full object-cover ring-[1.5px] ring-white", sizes[size])}
    />
);
