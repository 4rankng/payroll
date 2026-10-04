import * as React from "react"
import * as SwitchPrimitives from "@radix-ui/react-switch"

import { cn } from "@/lib/utils"

/**
 * Switch — UU PRO restyle (W7).
 *
 * Visual language follows the vendored UU v7 switch vocabulary on the W1
 * bridge: a larger rounded pill (`h-7 w-12`) with a sliding `size-5` white
 * indicator that transitions from utility-gray-200 to brand-solid when
 * activated. Focus uses the UU outline-brand treatment matching button/input.
 *
 * The radix API (`checked`, `onCheckedChange`, `disabled`, `id`, `className`)
 * is unchanged, so call sites can still override the track color via the
 * `data-[state=checked]` / `data-[state=unchecked]` attribute selectors
 * (CronJobTable does exactly that).
 */
const Switch = React.forwardRef<
  React.ElementRef<typeof SwitchPrimitives.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitives.Root>
>(({ className, ...props }, ref) => (
  <SwitchPrimitives.Root
    className={cn(
      "peer relative inline-flex h-7 w-12 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent px-1 transition-all duration-150 ease-out outline-brand focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-brand-solid data-[state=unchecked]:bg-utility-gray-200",
      // The pill is 28px tall for visual balance, but touch targets need 44px.
      // A transparent pseudo-element grows the hit area to 28 + 8 + 8 = 44px
      // without changing how the switch looks or shifting surrounding content.
      "before:absolute before:-inset-y-2 before:inset-x-0 before:content-['']",
      className
    )}
    {...props}
    ref={ref}
  >
    <SwitchPrimitives.Thumb
      className={cn(
        "pointer-events-none block size-5 rounded-full bg-white shadow-xs ring-0 transition-transform duration-150 ease-out data-[state=checked]:translate-x-5 data-[state=unchecked]:translate-x-0"
      )}
    />
  </SwitchPrimitives.Root>
))
Switch.displayName = SwitchPrimitives.Root.displayName

export { Switch }
