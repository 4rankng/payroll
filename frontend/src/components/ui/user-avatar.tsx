import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { generateAvatarUrl } from "@/utils/avatarHelpers";
import { cn } from "@/lib/utils";

interface UserAvatarProps {
  email?: string;
  name?: string;
  username?: string;
  cccd?: string;
  src?: string;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
}

const sizeClasses = {
  sm: 'h-6 w-6',
  md: 'h-8 w-8',
  lg: 'h-10 w-10',
  xl: 'h-12 w-12'
};

export function UserAvatar({
  name,
  src,
  size = 'md',
  className
}: UserAvatarProps) {
  // Without an uploaded photo every account falls back to the same neutral
  // profile glyph — never initials, never a portrait. Identity comes from the
  // name rendered next to the avatar.
  const avatarUrl = src || generateAvatarUrl();

  return (
    <Avatar className={cn(sizeClasses[size], className)}>
      <AvatarImage
        src={avatarUrl}
        alt={name ? `Ảnh đại diện của ${name}` : 'Ảnh đại diện'}
        className="object-cover"
      />
      <AvatarFallback
        aria-hidden="true"
        className="border border-primary/10 bg-primary/5"
      />
    </Avatar>
  );
}
