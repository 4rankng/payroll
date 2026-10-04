import * as React from "react"
import * as CheckboxPrimitive from "@radix-ui/react-checkbox"
import { Check } from "lucide-react"

import { cn } from "@/lib/utils"

// UU PRO restyle (W7): white surface + gray-300 border at rest; checked fills
// brand-solid with the check in white (vendored UU v7 checkbox vocabulary on
// the W1 bridge). The border idiom is kept instead of UU's ring so caller
// border-* tints keep overriding through the merge (BCCUploadModal). Focus
// uses the UU outline-brand treatment matching button/input.
const Checkbox = React.forwardRef<
  React.ElementRef<typeof CheckboxPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>
>(({ className, ...props }, ref) => (
  <CheckboxPrimitive.Root
    ref={ref}
    className={cn(
      "peer h-4 w-4 shrink-0 rounded-sm border border-utility-gray-300 bg-card shadow-xs outline-brand transition duration-100 ease-linear focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-brand-solid data-[state=checked]:bg-brand-solid data-[state=checked]:text-white",
      className
    )}
    {...props}
  >
    <CheckboxPrimitive.Indicator
      className={cn("flex items-center justify-center text-current")}
    >
      <Check className="h-4 w-4" />
    </CheckboxPrimitive.Indicator>
  </CheckboxPrimitive.Root>
))
Checkbox.displayName = CheckboxPrimitive.Root.displayName

export { Checkbox }
