import { CheckCircle, Clock, XCircle, Info } from 'lucide-react';
import { STATUS_LABELS } from '../types';

export const getStatusColor = (status: string) => {
  switch (status) {
    case 'active': 
      return 'bg-green-50 border-green-200 text-green-800 dark:bg-green-950 dark:border-green-800 dark:text-green-200';
    case 'pending_approval': 
      return 'bg-yellow-50 border-yellow-200 text-yellow-800 dark:bg-yellow-950 dark:border-yellow-800 dark:text-yellow-200';
    case 'rejected': 
      return 'bg-red-50 border-red-200 text-red-800 dark:bg-red-950 dark:border-red-800 dark:text-red-200';
    default: 
      return 'bg-muted border-muted-foreground/20 text-muted-foreground';
  }
};

export const getStatusIcon = (status: string) => {
  switch (status) {
    case 'active': 
      return CheckCircle;
    case 'pending_approval': 
      return Clock;
    case 'rejected': 
      return XCircle;
    default: 
      return Info;
  }
};

export const getStatusBadge = (status: string) => {
  const colors = {
    active: 'bg-green-100 text-green-800 border-green-200',
    pending_approval: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    rejected: 'bg-red-100 text-red-800 border-red-200',
    inactive: 'bg-muted text-gray-800 border-border'
  };
  
  return colors[status as keyof typeof colors] || colors.inactive;
};

export const getStatusLabel = (status: string) => {
  return STATUS_LABELS[status as keyof typeof STATUS_LABELS] || status;
};