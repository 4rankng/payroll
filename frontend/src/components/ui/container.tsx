import * as React from "react"
import { cn } from "@/lib/utils"

interface ContainerProps extends React.HTMLAttributes<HTMLDivElement> {
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  padding?: 'none' | 'sm' | 'md' | 'lg'
}

export const Container = React.forwardRef<HTMLDivElement, ContainerProps>(
  ({ className, size = 'lg', padding = 'md', ...props }, ref) => {
    const sizeClasses = {
      sm: 'max-w-sm mx-auto',
      md: 'max-w-3xl mx-auto',
      lg: 'max-w-5xl mx-auto',
      xl: 'max-w-7xl mx-auto',
      full: 'w-full',
    }

    const paddingClasses = {
      none: '',
      sm: 'px-4 sm:px-6',
      md: 'px-4 sm:px-6 md:px-8',
      lg: 'px-6 sm:px-8 md:px-10',
    }

    return (
      <div
        ref={ref}
        className={cn(sizeClasses[size], paddingClasses[padding], className)}
        {...props}
      />
    )
  }
)
Container.displayName = "Container"

export { Container }

