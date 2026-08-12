import { describe, expect, it } from "vitest";

import { getErrorDetails, getErrorMessage } from "./error-handler";

describe("getErrorMessage", () => {
  it("does not expose an upstream HTML gateway page", () => {
    const upstreamHTML =
      "<html><head><title>502 Bad Gateway</title></head><body>nginx</body></html>";

    expect(getErrorMessage(upstreamHTML)).toBe(
      "Máy chủ tạm thời không phản hồi. Vui lòng thử lại sau ít phút.",
    );
    expect(getErrorDetails(upstreamHTML).message).not.toContain("<html");
  });
});
