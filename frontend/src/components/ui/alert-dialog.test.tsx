import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogTrigger,
} from './alert-dialog';

function ConfirmDialog({ variant }: { variant?: 'destructive' }) {
  return (
    <AlertDialog>
      <AlertDialogTrigger>Xóa nhân viên</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogTitle>Xóa nhân viên?</AlertDialogTitle>
        <AlertDialogDescription>Hành động này không thể hoàn tác.</AlertDialogDescription>
        <AlertDialogFooter>
          <AlertDialogCancel>Hủy</AlertDialogCancel>
          <AlertDialogAction variant={variant}>Xác nhận</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

const openConfirm = () => fireEvent.click(screen.getByRole('button', { name: 'Xóa nhân viên' }));

describe('alert dialog contract', () => {
  it('names and describes the confirmation for screen readers', () => {
    render(<ConfirmDialog />);
    openConfirm();
    expect(screen.getByRole('alertdialog', { name: 'Xóa nhân viên?' })).toHaveAccessibleDescription('Hành động này không thể hoàn tác.');
  });

  it('renders the UU card surface inside the portal without daisyUI-scoped classes', () => {
    render(<ConfirmDialog />);
    openConfirm();
    const dialog = screen.getByRole('alertdialog');
    expect(dialog).toHaveClass('bg-card', 'text-card-foreground', 'border-utility-gray-200', 'shadow-2xl');
    expect(dialog.className).not.toContain('ct-');
    expect(dialog.className).not.toContain('base-100');
  });

  it('styles the default action with the UU brand solid and keeps the 44px target', () => {
    render(<ConfirmDialog />);
    openConfirm();
    const action = screen.getByRole('button', { name: 'Xác nhận' });
    expect(action).toHaveClass('bg-brand-solid', 'text-white', 'shadow-xs-skeumorphic', 'min-h-11');
    expect(screen.getByRole('button', { name: 'Hủy' })).toHaveClass('border-utility-gray-200', 'bg-card', 'min-h-11');
  });

  it('maps the destructive variant to the bridge error family', () => {
    render(<ConfirmDialog variant="destructive" />);
    openConfirm();
    expect(screen.getByRole('button', { name: 'Xác nhận' })).toHaveClass('bg-error-solid', 'outline-error');
  });

  it('passes overlayClassName to the overlay and closes from the cancel button', () => {
    render(
      <AlertDialog>
        <AlertDialogTrigger>Xóa</AlertDialogTrigger>
        <AlertDialogContent overlayClassName="bg-fuchsia-900/70">
          <AlertDialogTitle>Khóa sổ?</AlertDialogTitle>
          <AlertDialogFooter>
            <AlertDialogCancel>Để sau</AlertDialogCancel>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Xóa' }));
    expect(document.querySelector('.bg-fuchsia-900\\/70')).not.toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Để sau' }));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });
});
