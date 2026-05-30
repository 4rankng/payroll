import { createContext, useContext, ReactNode } from 'react';
import { useTransactionMetadata } from '@/hooks/transactions/useTransactionMetadata';
import { useAccountMetadata } from '@/hooks/ledger/useAccountMetadata';
import type { TransactionMetadata } from '@/services/api/transaction.service';
import type { AccountMetadata } from '@/types/api/financial.types';

interface MetadataContextValue {
  // Transaction metadata
  transactionMetadata: TransactionMetadata | undefined;
  isLoadingTransactionMetadata: boolean;
  transactionMetadataError: Error | null;

  // Ledger account metadata
  ledgerAccountMetadata: AccountMetadata[] | undefined;
  isLoadingLedgerMetadata: boolean;
  ledgerMetadataError: Error | null;

  // Combined loading state
  isLoading: boolean;
  hasError: boolean;
}

const MetadataContext = createContext<MetadataContextValue | undefined>(undefined);

interface MetadataProviderProps {
  children: ReactNode;
}

/**
 * Provider that fetches and caches metadata from backend
 * Makes transaction types, statuses, and ledger accounts available globally
 */
export const MetadataProvider = ({ children }: MetadataProviderProps) => {
  // Fetch transaction metadata
  const {
    data: transactionMetadata,
    isLoading: isLoadingTransactionMetadata,
    error: transactionMetadataError,
  } = useTransactionMetadata();

  // Fetch ledger account metadata
  const {
    data: ledgerAccountMetadata,
    isLoading: isLoadingLedgerMetadata,
    error: ledgerMetadataError,
  } = useAccountMetadata();

  const isLoading = isLoadingTransactionMetadata || isLoadingLedgerMetadata;
  const hasError = !!(transactionMetadataError || ledgerMetadataError);

  const value: MetadataContextValue = {
    transactionMetadata,
    isLoadingTransactionMetadata,
    transactionMetadataError: transactionMetadataError as Error | null,

    ledgerAccountMetadata,
    isLoadingLedgerMetadata,
    ledgerMetadataError: ledgerMetadataError as Error | null,

    isLoading,
    hasError,
  };

  return (
    <MetadataContext.Provider value={value}>
      {children}
    </MetadataContext.Provider>
  );
};

/**
 * Hook to access metadata context
 * Throws error if used outside MetadataProvider
 */
export const useMetadata = () => {
  const context = useContext(MetadataContext);
  if (context === undefined) {
    throw new Error('useMetadata must be used within MetadataProvider');
  }
  return context;
};
