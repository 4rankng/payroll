// ALTCHA <altcha-widget> web component: JSX intrinsic element registration
// (the altcha package's own JSX types omit ref support) plus the typed element
// class for refs/imperative methods.
import type * as React from "react";
import type { AltchaWidgetElement } from "altcha/types/generic";

export type { AltchaWidgetElement };

declare global {
  namespace JSX {
    interface IntrinsicElements {
      "altcha-widget": React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement> & {
        /** Stringified challenge JSON (Configuration.challenge attribute). */
        challenge?: string;
        /** UI language code; "vi" is registered by importing altcha/i18n/vi. */
        language?: string;
        /** Verification trigger: "onload" solves and ticks itself on mount. */
        auto?: string;
      };
    }
  }
}
