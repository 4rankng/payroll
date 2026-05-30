import React, { useEffect, useState } from 'react';
import { initializeModalStore, setupModalStoreSync } from '@/lib/modal-state-manager';
import { discoverModalConfigs, getRegistryStats } from '@/lib/modal-registry-auto';

/**
 * Modal System Initialization Hook
 * Sets up the comprehensive deep linking system with single source of truth
 */

interface ModalSystemState {
  isInitialized: boolean;
  isLoading: boolean;
  error: string | null;
  stats?: {
    total: number;
    byCategory: Record<string, number>;
    byRole: Record<string, number>;
    withDeeplink: number;
    withParams: number;
  };
}

export function useModalSystemInit() {
  const [state, setState] = useState<ModalSystemState>({
    isInitialized: false,
    isLoading: true,
    error: null
  });

  useEffect(() => {
    let mounted = true;

    async function initializeSystem() {
      try {
        setState(prev => ({ ...prev, isLoading: true, error: null }));

        // Step 1: Discover all modal configurations
        const configs = await discoverModalConfigs();

        if (!mounted) return;


        // Step 2: Get system statistics
        const stats = await getRegistryStats();

        if (!mounted) return;


        // Step 3: Initialize modal store from URL
        await initializeModalStore();

        if (!mounted) return;

        // Step 4: Set up URL synchronization
        setupModalStoreSync();

        if (!mounted) return;

        setState({
          isInitialized: true,
          isLoading: false,
          error: null,
          stats
        });


      } catch (error) {
        if (!mounted) return;

        const errorMessage = error instanceof Error ? error.message : 'Unknown error';
        console.error('❌ Failed to initialize modal system:', error);

        setState({
          isInitialized: false,
          isLoading: false,
          error: errorMessage
        });
      }
    }

    initializeSystem();

    return () => {
      mounted = false;
    };
  }, []);

  return state;
}

/**
 * Provider component to initialize modal system
 */
interface ModalSystemProviderProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  errorFallback?: (error: string) => React.ReactNode;
}

export function ModalSystemProvider({
  children,
  fallback,
  errorFallback
}: ModalSystemProviderProps) {
  const { isInitialized, isLoading, error, stats } = useModalSystemInit();

  if (error) {
    if (errorFallback) {
      return errorFallback(error);
    }

    return React.createElement('div', {
      className: "flex items-center justify-center min-h-screen"
    }, React.createElement('div', {
      className: "text-center space-y-4 p-6"
    }, [
      React.createElement('div', {
        key: 'title',
        className: "text-red-500 text-xl font-semibold"
      }, 'Lỗi khởi tạo hệ thống modal'),
      React.createElement('div', {
        key: 'message',
        className: "text-muted-foreground max-w-md"
      }, error),
      React.createElement('button', {
        key: 'button',
        onClick: () => window.location.reload(),
        className: "px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
      }, 'Tải lại trang')
    ]));
  }

  if (isLoading || !isInitialized) {
    if (fallback) {
      return fallback;
    }

    return React.createElement('div', {
      className: "flex items-center justify-center min-h-screen"
    }, React.createElement('div', {
      className: "text-center space-y-4"
    }, [
      React.createElement('div', {
        key: 'spinner',
        className: "animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"
      }),
      React.createElement('div', {
        key: 'text',
        className: "text-muted-foreground"
      }, 'Đang khởi tạo hệ thống modal...')
    ]));
  }

  // Log stats in development
    if (process.env.NODE_ENV === 'development' && stats) {
      // Stats available in dev mode
    }

  return children;
}