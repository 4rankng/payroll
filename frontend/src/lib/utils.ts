import { clsx, type ClassValue } from "clsx"
import { extendTailwindMerge } from "tailwind-merge"

// UU focus rings emit `outline outline-2 outline-offset-2`: bare `outline`
// (outline-style) must not clobber the outline-width utilities during the
// merge. tailwind-merge's default config classifies bare `outline` as an
// outline-width shorthand, so reassign it to outline-style where it belongs.
const twMerge = extendTailwindMerge({
  extend: {
    classGroups: {
      // Custom group ids that re-group bare `outline` as outline-style so the
      // UU focus-ring triplet survives the merge. 'outline-width' is a custom
      // id (not in tailwind-merge v2's DefaultClassGroupIds typings) — the
      // cast-free suppression keeps the runtime behaviour the contract tests
      // pin.
      // @ts-expect-error custom outline-width group id missing from v2 typings
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
