import React from "react"
import { useTheme } from "next-themes"
import { Toaster as Sonner, toast as baseToast } from "sonner"

type ToasterProps = React.ComponentProps<typeof Sonner>

// UU PRO restyle (W9): toast surface = bg-card + ring-utility-gray-200 +
// shadow-lg + rounded-lg (the landed select-content recipe); copy on
// fg-primary / fg-tertiary; action = brand solid, cancel = the landed
// secondary-gray recipe. Icons color from the bridge families via the toast's
// data-type hook; info has no bridge ladder, so it keeps the W4 Tailwind sky
// interim. Hook points (toast/description/actionButton/cancelButton) keep
// their keys and the "group toast" / "toaster group" hook classes; the
// toast() function contract is untouched (engine-keep: sonner runtime stays).
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
            "group toast group-[.toaster]:bg-card group-[.toaster]:text-fg-primary group-[.toaster]:ring-1 group-[.toaster]:ring-utility-gray-200 group-[.toaster]:shadow-lg group-[.toaster]:rounded-lg [&[data-type=success]_[data-icon]]:text-success-solid [&[data-type=error]_[data-icon]]:text-error-solid [&[data-type=warning]_[data-icon]]:text-warning-solid [&[data-type=info]_[data-icon]]:text-sky-700",
          title: "group-[.toast]:text-fg-primary",
          description: "group-[.toast]:text-fg-tertiary",
          actionButton:
            "group-[.toast]:bg-brand-solid group-[.toast]:text-fg-white group-[.toast]:shadow-xs-skeumorphic group-[.toast]:hover:bg-brand-solid_hover",
          cancelButton:
            "group-[.toast]:bg-utility-gray-100 group-[.toast]:text-utility-gray-700 group-[.toast]:hover:bg-utility-gray-200",
          closeButton:
            "group-[.toast]:!bg-card group-[.toast]:!border-utility-gray-200 group-[.toast]:text-fg-quaternary group-[.toast]:hover:!bg-utility-gray-50 group-[.toast]:!shadow-none",
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
  return baseToast(arg as string, opts)
}

export { Toaster, toast }
