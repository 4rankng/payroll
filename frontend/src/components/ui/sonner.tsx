import React from "react"
import { useTheme } from "next-themes"
import { Toaster as Sonner, toast as baseToast } from "sonner"

type ToasterProps = React.ComponentProps<typeof Sonner>

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme = "system" } = useTheme()

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
      duration={3000}
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:bg-background group-[.toaster]:text-foreground group-[.toaster]:border-border group-[.toaster]:shadow-sm",
          description: "group-[.toast]:text-muted-foreground",
          actionButton:
            "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
        },
        duration: 3000,
      }}
      {...props}
    />
  )
}

type ToastArgObject = {
  title: React.ReactNode
  description?: React.ReactNode
  variant?: string
  duration?: number
  [key: string]: unknown
}

function isToastArgObject(value: unknown): value is ToastArgObject {
  return !!value && typeof value === 'object' && 'title' in (value as Record<string, unknown>)
}

const toast = (arg: unknown, opts?: { duration?: number; className?: string; [key: string]: unknown }) => {
  if (isToastArgObject(arg)) {
    const { title, description, variant, duration, className, ...rest } = arg
    const normalize = (v: unknown) => {
      if (React.isValidElement(v) || typeof v === 'string' || typeof v === 'number') return v
      if (v == null) return undefined
      try { return JSON.stringify(v) } catch { return String(v) }
    }
    const normalizedTitle = normalize(title)
    const normalizedDescription = normalize(description)
    const finalOpts: Record<string, unknown> = {
      ...rest,
      description: normalizedDescription,
      duration: duration ?? opts?.duration,
    }
    const variantClass = variant ? `toast-variant-${variant}` : ''
    finalOpts.className = [className, opts?.className, variantClass].filter(Boolean).join(' ') || undefined
    return baseToast(normalizedTitle as React.ReactNode, finalOpts)
  }
  return baseToast(arg as any, opts)
}

export { Toaster, toast }
