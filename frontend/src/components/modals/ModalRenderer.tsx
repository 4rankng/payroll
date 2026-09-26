import { Suspense, useEffect, useState } from 'react';
import { useModalStore } from '@/lib/modal-state-manager';
import { getModalRegistryEntry } from '@/lib/modal-registry-auto';
import type { ModalRegistryEntry } from '@/types/modal-config.types';
import { Skeleton } from '@/components/ui/skeleton';

/**
 * Centralized Modal Renderer
 * Automatically discovers and renders modals based on current state
 * Single source of truth with no race conditions
 */

interface ModalRendererProps {
  className?: string;
}

function ModalSkeleton() {
  return (
    <div className="fixed inset-0 z-50 bg-card/80 backdrop-blur-sm">
      <div className="fixed inset-y-0 right-0 h-full w-full border-l bg-card shadow-sm sm:max-w-sm">
        <div className="flex flex-col h-full">
          <div className="p-6 border-b">
            <Skeleton className="h-6 w-48 mb-2" />
            <Skeleton className="h-4 w-32" />
          </div>
          <div className="flex-1 p-6 space-y-4">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-4 w-1/2" />
          </div>
          <div className="p-6 border-t">
            <div className="flex gap-3">
              <Skeleton className="h-10 flex-1" />
              <Skeleton className="h-10 flex-1" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function ModalLoader({ modalId, params }: { modalId: string; params: Record<string, unknown> }) {
  const [ModalComponent, setModalComponent] = useState<React.ComponentType<Record<string, unknown>> | null>(null);
  const [registryEntry, setRegistryEntry] = useState<ModalRegistryEntry | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { setError: setStoreError, setLoading } = useModalStore();

  useEffect(() => {
    let mounted = true;

    async function loadModal() {
      try {
        setLoading(true);
        setError(null);

        const entry = await getModalRegistryEntry(modalId);
        if (!entry) {
          throw new Error(`Modal not found: ${modalId}`);
        }

        if (!mounted) return;

        setRegistryEntry(entry);

        // Load the modal component
        const module = await entry.loader();
        if (!mounted) return;

        if (!module.default) {
          throw new Error(`Modal component not found: ${modalId}`);
        }

        setModalComponent(() => module.default);
      } catch (err) {
        if (!mounted) return;

        const errorMessage = err instanceof Error ? err.message : 'Failed to load modal';
        setError(errorMessage);
        setStoreError(errorMessage);
        console.error('Failed to load modal:', modalId, err);
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    }

    loadModal();

    return () => {
      mounted = false;
    };
  }, [modalId, setLoading, setStoreError]);

  if (error) {
    return (
      <div className="fixed inset-0 z-50 bg-card/80 backdrop-blur-sm">
        <div className="fixed inset-y-0 right-0 h-full w-full border-l bg-card shadow-sm sm:max-w-sm">
          <div className="flex items-center justify-center h-full p-6">
            <div className="text-center space-y-4">
              <div className="text-red-600 text-lg font-medium">Lỗi tải modal</div>
              <div className="text-muted-foreground text-sm">{error}</div>
              <button
                onClick={() => useModalStore.getState().closeModal()}
                className="px-4 py-2 bg-primary text-primary-foreground rounded-xl hover:bg-primary/90 transition-colors"
              >
                Đóng
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (!ModalComponent || !registryEntry) {
    return <ModalSkeleton />;
  }

  // Create props for the modal component
  const modalProps = {
    isOpen: true,
    onClose: () => useModalStore.getState().closeModal(),
    ...params
  };

  return <ModalComponent {...modalProps} />;
}

export function ModalRenderer({ className }: ModalRendererProps) {
  const { modalId, params, isLoading } = useModalStore();

  // Don't render anything if no modal is open
  if (!modalId) {
    return null;
  }

  return (
    <div className={className}>
      <Suspense fallback={<ModalSkeleton />}>
        {isLoading ? (
          <ModalSkeleton />
        ) : (
          <ModalLoader modalId={modalId} params={params} />
        )}
      </Suspense>
    </div>
  );
}

export default ModalRenderer;