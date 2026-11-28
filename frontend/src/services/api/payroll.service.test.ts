import { afterEach, describe, expect, it, vi } from "vitest";
import { apiClient } from "./client";
import { payrollService } from "./payroll.service";

const mockGetPayload = (payload: unknown) =>
  vi.spyOn(apiClient, "get").mockResolvedValue(payload as never);

/**
 * Payment-history payloads must expose an array of rows at `data` before the
 * pages render them — a drifted envelope (rows nested under data.rows /
 * data.data) previously reached the ledger builder unguarded and crashed the
 * partner payment-history page with "Cannot read properties of undefined".
 */
describe("payrollService history payload normalization", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  const record = {
    employee_id: 82,
    employee_name: "NGUYỄN VĂN A",
    employee_cccd: "031189014251",
    project_ids: [1],
    project_names: ["Dự án Việt Nam"],
    work_month: "2026-11",
    cycle: 2 as const,
    from_date: "2026-11-08",
    to_date: "2026-11-14",
    payment_date: "2026-11-17",
    total_amount: 1_998_000,
    transfers: [],
  };

  it("keeps a well-formed envelope untouched", async () => {
    mockGetPayload({
      status: "success",
      message: "ok",
      data: [record],
      pagination: { page: 1, pageSize: 50, totalPages: 1, totalRecords: 1 },
      summary: { total_amount: 1_998_000, transfer_count: 1, employee_count: 1 },
    });

    const result = await payrollService.getBankTransferHistories({ month: "2026-11" });

    expect(result.data).toHaveLength(1);
    expect(result.data[0].employee_name).toBe("NGUYỄN VĂN A");
    expect(result.pagination.totalRecords).toBe(1);
    expect(result.summary?.transfer_count).toBe(1);
  });

  it("unwraps rows nested under data.rows and lifts nested pagination/summary", async () => {
    mockGetPayload({
      status: "success",
      message: "",
      data: {
        rows: [record],
        pagination: { page: 2, pageSize: 50, totalPages: 3, totalRecords: 101 },
        summary: { total_amount: 4_000_000, transfer_count: 4, employee_count: 2 },
      },
    });

    const result = await payrollService.getBankTransferHistories({ month: "2026-11" });

    expect(Array.isArray(result.data)).toBe(true);
    expect(result.data[0].employee_id).toBe(82);
    expect(result.pagination.totalRecords).toBe(101);
    expect(result.summary?.employee_count).toBe(2);
  });

  it("unwraps rows nested under data.data when the rows key is absent", async () => {
    mockGetPayload({
      status: "success",
      message: "",
      data: { data: [record], pagination: { page: 1, pageSize: 50, totalPages: 1, totalRecords: 1 } },
    });

    const result = await payrollService.getPaymentHistories({ page: 1, pageSize: 20 });

    expect(Array.isArray(result.data)).toBe(true);
    expect(result.data[0].employee_name).toBe("NGUYỄN VĂN A");
    expect(result.pagination.totalRecords).toBe(1);
  });

  it("degrades a missing or null payload to an empty ledger", async () => {
    mockGetPayload(undefined);

    const result = await payrollService.getBankTransferHistories({ month: "2026-11" });

    expect(Array.isArray(result.data)).toBe(true);
    expect(result.data).toHaveLength(0);
  });
});
