import { describe, expect, it } from "vitest"

import { getSidebarCollapsedState } from "./sidebar"

describe("getSidebarCollapsedState", () => {
  it("uses the mobile sheet state on mobile even when the desktop sidebar is open", () => {
    expect(
      getSidebarCollapsedState({
        isMobile: true,
        open: true,
        openMobile: false,
      }),
    ).toBe(true)
  })

  it("reports an open mobile sheet as expanded regardless of desktop state", () => {
    expect(
      getSidebarCollapsedState({
        isMobile: true,
        open: false,
        openMobile: true,
      }),
    ).toBe(false)
  })

  it("uses the desktop sidebar state on desktop even when the mobile sheet is open", () => {
    expect(
      getSidebarCollapsedState({
        isMobile: false,
        open: false,
        openMobile: true,
      }),
    ).toBe(true)
  })

  it("reports an open desktop sidebar as expanded regardless of mobile state", () => {
    expect(
      getSidebarCollapsedState({
        isMobile: false,
        open: true,
        openMobile: false,
      }),
    ).toBe(false)
  })
})
