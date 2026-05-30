import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { generateAvatarUrl, getUserInitials } from "@/utils/avatarHelpers";
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

const textSizes = {
  sm: 'typography-body-small',
  md: 'typography-body-medium',
  lg: 'typography-body-large',
  xl: 'typography-title-large'
};

export function UserAvatar({
  email,
  name,
  username,
  cccd,
  src,
  size = 'md',
  className
}: UserAvatarProps) {
  const seed = username || cccd || name || 'default';
  const generatedUrl = generateAvatarUrl(seed);
  const avatarUrl = src || generatedUrl;
  const initials = getUserInitials(name);

  return (
    <Avatar className={cn(sizeClasses[size], className)}>
      <AvatarImage 
        src={avatarUrl} 
        alt={name || 'User avatar'}
        className="object-cover"
      />
      <AvatarFallback className={cn(
        'bg-white text-black border border-gray-300 font-medium',
        textSizes[size]
      )}>
        {initials}
      </AvatarFallback>
    </Avatar>
  );
}
