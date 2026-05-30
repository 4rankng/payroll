import type { ModalConfig, ModalRegistryEntry } from '@/types/modal-config.types';
import type { ComponentType } from 'react';

/**
 * Auto-Discovery Modal Registry System
 * Single source of truth for all modals with automatic discovery from modalConfig exports
 */

// Dynamic imports for all modal components with their configurations
const modalModules = import.meta.glob('/src/components/**/*{Sheet,Modal}.tsx', {
  eager: false,
}) as Record<string, () => Promise<{ default: ComponentType<Record<string, unknown>>; modalConfig?: ModalConfig }>>;

// Cache for loaded modal configurations
const modalConfigCache = new Map<string, ModalConfig>();
const modalRegistryCache = new Map<string, ModalRegistryEntry>();

/**
 * Auto-discover and load modal configurations from file exports
 */
export async function discoverModalConfigs(): Promise<Map<string, ModalConfig>> {
  const configs = new Map<string, ModalConfig>();

  // Load all modal modules and extract their modalConfig exports
  const loadPromises = Object.entries(modalModules).map(async ([filePath, loader]) => {
    try {
      const module = await loader();
      if (module.modalConfig && typeof module.modalConfig === 'object') {
        const config = module.modalConfig as ModalConfig;

        // Validate configuration
        if (config.id && config.permissions && config.deeplink) {
          configs.set(config.id, {
            ...config,
            // Add file path for debugging
            filePath: filePath.replace('/src/', '@/')
          } as ModalConfig & { filePath: string });

          // Cache the config
          modalConfigCache.set(config.id, config);
        } else {
          console.warn(`Invalid modal config in ${filePath}:`, config);
        }
      }
    } catch (error) {
      console.warn(`Failed to load modal config from ${filePath}:`, error);
    }
  });

  await Promise.all(loadPromises);
  return configs;
}

/**
 * Build complete modal registry with lazy loaders
 */
export async function buildModalRegistry(): Promise<Map<string, ModalRegistryEntry>> {
  const registry = new Map<string, ModalRegistryEntry>();
  const configs = await discoverModalConfigs();

  for (const [modalId, config] of configs) {
    // Find the corresponding module loader
    const moduleEntry = Object.entries(modalModules).find(([filePath]) => {
      // Match by config file path or modal ID pattern
      return filePath.includes(modalId.replace(/_/g, '-')) ||
             filePath.includes(modalId.replace(/_/g, ''));
    });

    if (moduleEntry) {
      const [filePath, loader] = moduleEntry;

      registry.set(modalId, {
        ...config,
        loader: async () => {
          const module = await loader();
          return {
            default: module.default,
            modalConfig: module.modalConfig || config
          };
        },
        filePath: filePath.replace('/src/', '@/')
      });

      // Cache the registry entry
      modalRegistryCache.set(modalId, registry.get(modalId)!);
    }
  }

  return registry;
}

/**
 * Get modal configuration by ID (with caching)
 */
export async function getModalConfig(modalId: string): Promise<ModalConfig | null> {
  // Check cache first
  if (modalConfigCache.has(modalId)) {
    return modalConfigCache.get(modalId)!;
  }

  // Discover and cache configs if not found
  const configs = await discoverModalConfigs();
  return configs.get(modalId) || null;
}

/**
 * Get modal registry entry by ID (with caching)
 */
export async function getModalRegistryEntry(modalId: string): Promise<ModalRegistryEntry | null> {
  // Check cache first
  if (modalRegistryCache.has(modalId)) {
    return modalRegistryCache.get(modalId)!;
  }

  // Build registry if not cached
  const registry = await buildModalRegistry();
  return registry.get(modalId) || null;
}

/**
 * Get all available modal IDs
 */
export async function getAllModalIds(): Promise<string[]> {
  const configs = await discoverModalConfigs();
  return Array.from(configs.keys()).sort();
}

/**
 * Validate modal parameters against configuration
 */
export async function validateModalParams(
  modalId: string,
  params: Record<string, unknown>
): Promise<boolean> {
  const config = await getModalConfig(modalId);
  if (!config) return false;

  // Use custom validation if provided
  if (config.deeplink.validateParams) {
    return config.deeplink.validateParams(params);
  }

  // Default validation: check required params exist
  if (config.deeplink.params && config.deeplink.params.length > 0) {
    return config.deeplink.params.every(param =>
      params[param] !== undefined && params[param] !== null && params[param] !== ''
    );
  }

  return true;
}

/**
 * Get modal metadata for navigation
 */
export async function getModalMetadata(modalId: string) {
  const config = await getModalConfig(modalId);
  if (!config) return null;

  return {
    id: modalId,
    name: config.name || modalId,
    description: config.description || '',
    category: config.category || 'general',
    requiresAuth: config.requiresAuth ?? true,
    roles: config.permissions.roles,
    hasParams: config.deeplink.params && config.deeplink.params.length > 0,
    params: config.deeplink.params || [],
    example: config.deeplink.example
  };
}

/**
 * Development helper: Get registry statistics
 */
export async function getRegistryStats() {
  const configs = await discoverModalConfigs();
  const configArray = Array.from(configs.values());

  return {
    total: configArray.length,
    byCategory: configArray.reduce((acc, config) => {
      const category = config.category || 'general';
      acc[category] = (acc[category] || 0) + 1;
      return acc;
    }, {} as Record<string, number>),
    byRole: configArray.reduce((acc, config) => {
      config.permissions.roles.forEach(role => {
        acc[role] = (acc[role] || 0) + 1;
      });
      return acc;
    }, {} as Record<string, number>),
    withDeeplink: configArray.filter(c => c.deeplink.enabled).length,
    withParams: configArray.filter(c => c.deeplink.params && c.deeplink.params.length > 0).length
  };
}

/**
 * Clear all caches (useful for development/testing)
 */
export function clearModalCache() {
  modalConfigCache.clear();
  modalRegistryCache.clear();
}
