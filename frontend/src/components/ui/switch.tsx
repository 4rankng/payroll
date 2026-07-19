import * as React from "react"
import * as SwitchPrimitives from "@radix-ui/react-switch"

import { cn } from "@/lib/utils"

/**
 * Switch — based on Tailkit `a-c-form-switches-01` (Simple).
 *
 * Visual language adopted from the Tailkit simple form switch: a larger
 * rounded pill (`h-7 w-12`) with a sliding `size-5` circular indicator that
 * transitions smoothly from the muted input color to the brand primary when
 * activated, plus a soft primary-tinted focus ring with offset.
 *
 * Colors intentionally map to the project's shadcn HSL tokens (`bg-input`,
 * `bg-primary`, `ring-ring`) rather than Tailkit's raw `secondary-300` /
 * `emerald-500` so the switch stays consistent with the rest of the design
 * system across admin / partner / employee themes.
 *
 * The radix API (`checked`, `onCheckedChange`, `disabled`, `id`, `className`)
 * is unchanged, so call sites can still override the track color via the
 * `data-[state=checked]` / `data-[state=unchecked]` attribute selectors.
 */
const Switch = React.forwardRef<
  React.ElementRef<typeof SwitchPrimitives.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitives.Root>
>(({ className, ...props }, ref) => (
  <SwitchPrimitives.Root
    className={cn(
      "peer inline-flex h-7 w-12 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent px-1 transition-all duration-150 ease-out outline-none focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=unchecked]:bg-input",
      className
    )}
    {...props}
    ref={ref}
  >
    <SwitchPrimitives.Thumb
      className={cn(
        "pointer-events-none block size-5 rounded-full bg-white shadow-sm ring-0 transition-transform duration-150 ease-out data-[state=checked]:translate-x-5 data-[state=unchecked]:translate-x-0"
      )}
    />
  </SwitchPrimitives.Root>
))
Switch.displayName = SwitchPrimitives.Root.displayName

export { Switch }
