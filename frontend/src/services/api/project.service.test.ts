import { afterEach, describe, expect, it, vi } from "vitest";
import { apiClient } from "./client";
import { projectService } from "./project.service";

describe("projectService.getProjects", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("uses the backend sort parameter contract for desktop and mobile lists", async () => {
    const get = vi
      .spyOn(apiClient, "get")
      .mockResolvedValue({ status: "success", data: [], message: "ok" });

    await projectService.getProjects({
      page: 1,
      pageSize: 20,
      sortBy: "name",
      sortOrder: "asc",
    });

    expect(get).toHaveBeenCalledWith(
      expect.stringContaining(
        "page=1&pageSize=20&sort_by=name&sort_order=asc",
      ),
    );
    expect(get.mock.calls[0][0]).not.toContain("sortBy=");
    expect(get.mock.calls[0][0]).not.toContain("sortOrder=");
  });
});
