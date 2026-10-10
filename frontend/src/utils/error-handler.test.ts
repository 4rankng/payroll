import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/components/ui/sonner", () => ({ toast: vi.fn() }));

import { toast } from "@/components/ui/sonner";
import { getErrorDetails, getErrorMessage, showErrorNotification } from "./error-handler";

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

describe("showErrorNotification", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // A rejected upload (unsupported template) is surfaced only through the
  // toast, so it must carry the server's Vietnamese reason and stay closable.
  it("forwards the server message and makes the toast dismissible", () => {
    showErrorNotification(
      "Mẫu file chấm công không đúng. Vui lòng tải lên tệp Excel theo đúng mẫu chấm công của dự án.",
    );

    expect(toast).toHaveBeenCalledTimes(1);
    const options = vi.mocked(toast).mock.calls[0][0] as {
      description?: string;
      closeButton?: boolean;
    };
    expect(options.description).toBe(
      "Mẫu file chấm công không đúng. Vui lòng tải lên tệp Excel theo đúng mẫu chấm công của dự án.",
    );
    expect(options.closeButton).toBe(true);
  });
});
