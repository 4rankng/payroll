import * as React from "react"
import { cn } from "@/lib/utils"
import { Card, CardContent, CardHeader } from "@/components/ui/card"

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

export const LoadingSpinner = ({ size = 'md', className }: LoadingSpinnerProps) => {
  const sizeClasses = {
    sm: 'h-4 w-4',
    md: 'h-6 w-6',
    lg: 'h-8 w-8',
  }

  return (
    <div
      className={cn(
        'animate-spin rounded-full border-2 border-current border-t-transparent',
        sizeClasses[size],
        className
      )}
    />
  )
}

interface LoadingCardProps {
  className?: string
  lines?: number
}

export const LoadingCard = ({ className, lines = 3 }: LoadingCardProps) => (
  <Card className={cn('animate-pulse', className)}>
    <CardHeader>
      <div className="h-4 skeleton-shimmer rounded w-3/4"></div>
    </CardHeader>
    <CardContent className="space-y-2">
      <div className="h-8 skeleton-shimmer rounded w-1/2 mb-2"></div>
      {Array.from({ length: lines }).map((_, i) => (
        <div key={i} className="h-3 skeleton-shimmer rounded w-full"></div>
      ))}
    </CardContent>
  </Card>
)

interface LoadingButtonProps {
  className?: string
  size?: 'sm' | 'md' | 'lg'
}

export const LoadingButton = ({ className, size = 'md' }: LoadingButtonProps) => {
  const sizeClasses = {
    sm: 'h-9 px-3',
    md: 'h-11 px-4',
    lg: 'h-12 px-8',
  }

  return (
    <div
      className={cn(
        'inline-flex items-center justify-center gap-2 rounded-md bg-muted animate-pulse',
        sizeClasses[size],
        className
      )}
    >
      <LoadingSpinner size="sm" />
      <span className="typography-body-medium">Loading...</span>
    </div>
  )
}

interface LoadingTableProps {
  rows?: number
  columns?: number
  className?: string
}

export const LoadingTable = ({ rows = 5, columns = 4, className }: LoadingTableProps) => (
  <div className={cn('space-y-4', className)}>
    {/* Header */}
    <div className="grid gap-4" style={{ gridTemplateColumns: `repeat(${columns}, 1fr)` }}>
      {Array.from({ length: columns }).map((_, i) => (
        <div key={i} className="h-4 skeleton-shimmer rounded"></div>
      ))}
    </div>

    {/* Rows */}
    {Array.from({ length: rows }).map((_, rowIndex) => (
      <div key={rowIndex} className="grid gap-4" style={{ gridTemplateColumns: `repeat(${columns}, 1fr)` }}>
        {Array.from({ length: columns }).map((_, colIndex) => (
          <div key={colIndex} className="h-6 skeleton-shimmer rounded"></div>
        ))}
      </div>
    ))}
  </div>
)

LoadingSpinner.displayName = "LoadingSpinner"
LoadingCard.displayName = "LoadingCard"
LoadingButton.displayName = "LoadingButton"
LoadingTable.displayName = "LoadingTable"

