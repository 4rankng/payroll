import { fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import UserDetailsSheet from "./UserDetailsSheet";
import type { User } from "@/types/user";

vi.mock("./templates/SlideSheetTemplate", () => ({
  SlideSheetTemplate: ({ children, footer }: { children: ReactNode; footer: ReactNode }) => (
    <div>
      {children}
      {footer}
    </div>
  ),
}));

vi.mock("@/components/users/details/UserHeader", () => ({
  UserHeader: () => <div>Thông tin người dùng</div>,
}));

vi.mock("@/components/users/details/UserActions", () => ({
  UserActions: ({ isEditing, onEdit, onSave }: { isEditing: boolean; onEdit: () => void; onSave: () => void }) => (
    isEditing
      ? <button type="button" onClick={onSave}>Lưu</button>
      : <button type="button" onClick={onEdit}>Chỉnh sửa</button>
  ),
}));

vi.mock("@/hooks/api/useUsers", () => ({
  useUserActivities: () => ({ data: undefined, isLoading: false }),
}));

vi.mock("@/components/modals/ResetPasswordModal", () => ({ ResetPasswordModal: () => null }));
vi.mock("@/components/ui/confirm-dialog", () => ({ ConfirmDialog: () => null }));

const createUser = (overrides: Partial<User> = {}): User => ({
  id: 1,
  email: "admin@example.com",
  username: "admin",
  fullname: "Admin User",
  mobile: "0901234567",
  role: "admin",
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

const renderSheet = (user: User, onUpdate = vi.fn()) => {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <UserDetailsSheet
        user={user}
        isOpen
        onClose={vi.fn()}
        onUpdate={onUpdate}
        onDelete={vi.fn()}
      />
    </QueryClientProvider>,
  );
  return onUpdate;
};

describe("UserDetailsSheet mobile ownership", () => {
  it("shows and updates users.mobile for an Admin account", () => {
    const onUpdate = renderSheet(createUser());

    expect(screen.getByText("0901234567")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Chỉnh sửa" }));
    fireEvent.change(screen.getByLabelText("Số điện thoại"), { target: { value: "+84 912 345 678" } });
    fireEvent.click(screen.getByRole("button", { name: "Lưu" }));

    expect(onUpdate).toHaveBeenCalledWith(1, expect.objectContaining({ mobile: "+84 912 345 678" }));
  });

  it("does not expose users.mobile for an Employee account", () => {
    renderSheet(createUser({ role: "employee", mobile: undefined }));

    expect(screen.queryByText("Số điện thoại")).not.toBeInTheDocument();
  });
});
