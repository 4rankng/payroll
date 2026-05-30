import React from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { AlertTriangle, Info } from 'lucide-react';
import type { ValidationResult } from '../../types';

interface ValidationAlertProps {
  validation?: ValidationResult;
}

export function ValidationAlert({ validation }: ValidationAlertProps) {
  if (!validation) return null;

  return (
    <>
      {/* Validation Errors */}
      {validation.errors.length > 0 && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            {validation.errors[0]}
          </AlertDescription>
        </Alert>
      )}

      {/* Validation Warnings */}
      {validation.warnings && validation.warnings.length > 0 && (
        <Alert>
          <Info className="h-4 w-4" />
          <AlertDescription>
            {validation.warnings[0]}
          </AlertDescription>
        </Alert>
      )}
    </>
  );
}