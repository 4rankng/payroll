import { describe, expect, it } from "vitest";

import { projectEmployeesKey } from "./queryKeys";

describe("projectEmployeesKey", () => {
  it("keeps backend search terms in the project employee cache identity", () => {
    const unfilteredKey = projectEmployeesKey(7, {
      page: 1,
      pageSize: 50,
    });
    const searchedKey = projectEmployeesKey(7, {
      page: 1,
      pageSize: 50,
      search: "viet duy",
    });

    expect(searchedKey).not.toEqual(unfilteredKey);
    expect(searchedKey).toEqual([
      "projects",
      7,
      "employees",
      {
        page: 1,
        pageSize: 50,
        search: "viet duy",
      },
    ]);
  });
});
