// Shared modal types
import { ReactNode } from 'react';

export interface BaseModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export interface ModalProps extends BaseModalProps {
  title?: string;
  description?: string;
  children?: ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl';
}

export interface ConfirmModalProps extends BaseModalProps {
  title: string;
  description: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'default' | 'destructive';
  onConfirm: () => void;
}

export interface FormModalProps<T = unknown> extends BaseModalProps {
  title: string;
  initialData?: T;
  onSubmit: (data: T) => void | Promise<void>;
}

// Modal registry types
export interface ModalRegistryEntry {
  component: React.ComponentType<ModalProps>;
  defaultProps?: Partial<ModalProps>;
}

export interface ModalManager {
  isOpen: boolean;
  modalType: string | null;
  modalProps: Record<string, unknown>;
  openModal: (modalType: string, props?: Record<string, unknown>) => void;
  closeModal: () => void;
}