import * as RadixAvatar from "@radix-ui/react-avatar"
import * as React from "react"

import { cn } from "@/lib/utils"

/**
 * UU PRO restyle (vendored reference: `@/components/base/avatar`):
 * utility-gray surface with the UU inner contrast outline; compound
 * shadcn shape (Avatar / AvatarImage / AvatarFallback) unchanged.
 */
const Avatar = React.forwardRef<
  React.ElementRef<typeof RadixAvatar.Root>,
  React.ComponentPropsWithoutRef<typeof RadixAvatar.Root>
>(({ className, ...props }, ref) => (
  <RadixAvatar.Root
    ref={ref}
    className={cn(
      "relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full bg-utility-gray-100 outline outline-utility-gray-200 outline-1 -outline-offset-1",
      className,
    )}
    {...props}
  />
))
Avatar.displayName = RadixAvatar.Root.displayName

const AvatarImage = React.forwardRef<
  React.ElementRef<typeof RadixAvatar.Image>,
  React.ComponentPropsWithoutRef<typeof RadixAvatar.Image>
>(({ className, ...props }, ref) => (
  <RadixAvatar.Image
    ref={ref}
    className={cn("aspect-square h-full w-full object-cover", className)}
    {...props}
  />
))
AvatarImage.displayName = RadixAvatar.Image.displayName

const AvatarFallback = React.forwardRef<
  React.ElementRef<typeof RadixAvatar.Fallback>,
  React.ComponentPropsWithoutRef<typeof RadixAvatar.Fallback>
>(({ className, ...props }, ref) => (
  <RadixAvatar.Fallback
    ref={ref}
    className={cn(
      "flex h-full w-full items-center justify-center rounded-full bg-utility-gray-100 text-fg-quaternary",
      className,
    )}
    {...props}
  />
))
AvatarFallback.displayName = RadixAvatar.Fallback.displayName

export { Avatar, AvatarImage, AvatarFallback }
