import { ReactNode } from "react";

export interface SlideSheetTemplateAvatar {
  icon?: React.ComponentType<{ className?: string }>;
  text?: string;
  fallback?: ReactNode;
  custom?: ReactNode;
}

export interface SlideSheetTemplateProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  avatar?: SlideSheetTemplateAvatar;
  children: ReactNode;
  footer?: ReactNode;
  headerActions?: ReactNode;
  size?: 'default' | 'large' | 'full';
  className?: string;
  compact?: boolean;
}
