import { clsx, type ClassValue } from "clsx"
import { extendTailwindMerge } from "tailwind-merge"

// UU focus rings emit `outline outline-2 outline-offset-2`: bare `outline`
// (outline-style) must not clobber the outline-width utilities during the
// merge. tailwind-merge's default config classifies bare `outline` as an
// outline-width shorthand, so reassign it to outline-style where it belongs.
const twMerge = extendTailwindMerge({
  extend: {
    classGroups: {
      "outline-width": ["outline-0", "outline-1", "outline-2", "outline-4", "outline-8"],
      "outline-style": [
        "outline",
        "outline-none",
        "outline-hidden",
        "outline-dashed",
        "outline-dotted",
        "outline-double",
        "outline-solid",
      ],
    },
  },
})

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
