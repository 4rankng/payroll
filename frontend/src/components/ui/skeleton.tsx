import * as React from "react"

import { cn } from "@/lib/utils"

// UU PRO restyle: skeleton gray (utility-gray-100) instead of bg-muted.
function Skeleton({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("animate-pulse rounded-md bg-utility-gray-100", className)}
      {...props}
    />
  )
}

export { Skeleton }
