/**
 * JSX typing for the ALTCHA widget web component (npm `altcha`).
 * The package's generic types declare HTMLElementTagNameMap but not React's
 * JSX.IntrinsicElements, so a login-page render would fail type-check without
 * this declaration.
 */
import type { AltchaWidgetElement } from "altcha";

type AltchaWidgetAttributes = React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement> & {
  ref?: React.Ref<AltchaWidgetElement>;
  /** Challenge data or URL to fetch a challenge from. */
  challenge?: string;
  /** ISO alpha-2 language code (requires the matching i18n import). */
  language?: string;
  /** When verification triggers automatically. */
  auto?: "off" | "onfocus" | "onload" | "onsubmit";
  /** JSON-encoded advanced configuration (hideLogo, hideFooter, ...). */
  configuration?: string;
};

declare global {
  namespace React {
    namespace JSX {
      interface IntrinsicElements {
        "altcha-widget": AltchaWidgetAttributes;
      }
    }
  }
}

export {};
