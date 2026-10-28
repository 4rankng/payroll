import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("Input font-size ≥16px trên mobile (PAYR-25)", () => {
  const css = readFileSync(resolve(process.cwd(), "src/styles/utilities.css"), "utf8");

  it("mọi input/textarea/select ≥16px dưới 768px — chặn iOS Safari zoom khi focus", () => {
    expect(css).toMatch(
      /@media \(max-width: 767\.98px\)[\s\S]*?input,\s*textarea,\s*select\s*\{\s*font-size: 16px !important;/,
    );
  });

  it("không đụng font-size desktop: rule nằm trong media <768px", () => {
    const guardIndex = css.indexOf("font-size: 16px !important");
    const mediaStart = css.lastIndexOf("@media (max-width: 767.98px)", guardIndex);
    expect(mediaStart).toBeGreaterThan(-1);
    // Guard đứng trước utility tap-target (cùng media nhưng riêng rule)
    expect(css.indexOf("PAYR-24")).toBeGreaterThan(guardIndex);
  });
});
