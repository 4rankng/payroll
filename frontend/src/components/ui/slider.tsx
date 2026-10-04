import * as React from "react"
import * as SliderPrimitive from "@radix-ui/react-slider"

import { cn } from "@/lib/utils"

// UU PRO restyle (W7): track on the utility-gray ladder, range and thumb in
// brand-solid (vendored UU v7 slider vocabulary on the W1 bridge). Focus uses
// the UU outline-brand treatment matching button/input.
const Slider = React.forwardRef<
  React.ElementRef<typeof SliderPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SliderPrimitive.Root>
>(({ className, "aria-label": ariaLabel, "aria-labelledby": ariaLabelledBy, ...props }, ref) => (
  <SliderPrimitive.Root
    ref={ref}
    className={cn(
      "relative flex w-full touch-none select-none items-center",
      className
    )}
    {...props}
  >
    <SliderPrimitive.Track className="relative h-2 w-full grow overflow-hidden rounded-full bg-utility-gray-200">
      <SliderPrimitive.Range className="absolute h-full bg-brand-solid" />
    </SliderPrimitive.Track>
    {/* Radix does not forward ARIA label attributes from Root to the Thumb
        (which carries role="slider"), so they are passed explicitly here. */}
    <SliderPrimitive.Thumb
      aria-label={ariaLabel}
      aria-labelledby={ariaLabelledBy}
      className="block h-5 w-5 rounded-full border-2 border-brand-solid bg-brand-solid outline-brand transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 disabled:pointer-events-none disabled:opacity-50"
    />
  </SliderPrimitive.Root>
))
Slider.displayName = SliderPrimitive.Root.displayName

export { Slider }
