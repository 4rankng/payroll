/**
 * Smart Invalidation Service
 * Handles batched, efficient cache invalidation with debugging and monitoring
 */

import { QueryClient } from '@tanstack/react-query';
import {
  getInvalidationPatterns,
  resolveGroupReference,
  type InvalidationContext,
  type InvalidationPattern,
  InvalidationRegistry,
} from './invalidationRegistry';
import { QueryKeyUtils } from '@/lib/queryKeys';

/**
 * Invalidation strategy types
 */
export type InvalidationStrategy = 'immediate' | 'batched' | 'deferred';

/**
 * Invalidation options
 */
export interface InvalidationOptions {
  strategy?: InvalidationStrategy;
  refetchActive?: boolean;
  refetchInactive?: boolean;
  debug?: boolean;
  skipDuplicates?: boolean;
}

/**
 * Invalidation result for monitoring
 */
export interface InvalidationResult {
  mutationType: keyof typeof InvalidationRegistry;
  context: InvalidationContext;
  queryKeys: readonly unknown[][];
  duration: number;
  success: boolean;
  error?: Error;
}

/**
 * Default invalidation options
 */
const DEFAULT_OPTIONS: Required<InvalidationOptions> = {
  strategy: 'immediate',
  refetchActive: true,
  refetchInactive: false,
  debug: process.env.NODE_ENV === 'development',
  skipDuplicates: true,
};

/**
 * Smart Invalidation Service Class
 */
export class InvalidationService {
  private queryClient: QueryClient;
  private batchTimeout: NodeJS.Timeout | null = null;
  private batchedInvalidations: Set<string> = new Set();
  private invalidationHistory: InvalidationResult[] = [];

  constructor(queryClient: QueryClient) {
    this.queryClient = queryClient;
  }

  /**
   * Main invalidation method
   */
  async invalidate(
    mutationType: keyof typeof InvalidationRegistry,
    context: InvalidationContext = {},
    options: InvalidationOptions = {}
  ): Promise<InvalidationResult> {
    const startTime = performance.now();
    const finalOptions = { ...DEFAULT_OPTIONS, ...options };

    try {
      const patterns = getInvalidationPatterns(mutationType, context);
      const queryKeys = this.resolvePatterns(patterns, context);

      if (finalOptions.debug) {
        this.logInvalidation(mutationType, context, queryKeys);
      }

      switch (finalOptions.strategy) {
        case 'immediate':
          await this.immediateInvalidate(queryKeys, finalOptions);
          break;
        case 'batched':
          this.batchInvalidate(queryKeys, finalOptions);
          break;
        case 'deferred':
          this.deferInvalidate(queryKeys, finalOptions);
          break;
      }

      const duration = performance.now() - startTime;
      const result: InvalidationResult = {
        mutationType,
        context,
        queryKeys,
        duration,
        success: true,
      };

      this.recordInvalidation(result);
      return result;
    } catch (error) {
      const duration = performance.now() - startTime;
      const result: InvalidationResult = {
        mutationType,
        context,
        queryKeys: [],
        duration,
        success: false,
        error: error as Error,
      };

      this.recordInvalidation(result);
      throw error;
    }
  }

  /**
   * Resolve invalidation patterns to actual query keys
   */
  private resolvePatterns(
    patterns: InvalidationPattern[],
    context: InvalidationContext
  ): readonly unknown[][] {
    const resolvedKeys: readonly unknown[][] = [];

    for (const pattern of patterns) {
      if (typeof pattern === 'string') {
        // Group reference
        const groupKeys = resolveGroupReference(pattern, context);
        resolvedKeys.push(...groupKeys);
      } else if (typeof pattern === 'function') {
        // Dynamic pattern
        const result = pattern(context);
        if (Array.isArray(result[0])) {
          // Multiple keys
          resolvedKeys.push(...(result as readonly unknown[][]));
        } else {
          // Single key
          resolvedKeys.push(result as readonly unknown[]);
        }
      } else {
        // Static pattern
        resolvedKeys.push(pattern);
      }
    }

    return this.deduplicateKeys(resolvedKeys);
  }

  /**
   * Remove duplicate query keys
   */
  private deduplicateKeys(keys: readonly unknown[][]): readonly unknown[][] {
    const seen = new Set<string>();
    const unique: readonly unknown[][] = [];

    for (const key of keys) {
      const keyString = JSON.stringify(key);
      if (!seen.has(keyString)) {
        seen.add(keyString);
        unique.push(key);
      }
    }

    return unique;
  }

  /**
   * Immediate invalidation strategy
   */
  private async immediateInvalidate(
    queryKeys: readonly unknown[][],
    options: Required<InvalidationOptions>
  ): Promise<void> {
    const promises = queryKeys.map((queryKey) =>
      this.queryClient.invalidateQueries({
        queryKey,
        refetchType: options.refetchInactive ? 'all' : 'active',
      })
    );

    await Promise.all(promises);
  }

  /**
   * Batched invalidation strategy (debounced)
   */
  private batchInvalidate(
    queryKeys: readonly unknown[][],
    options: Required<InvalidationOptions>
  ): void {
    // Add keys to batch
    queryKeys.forEach((key) => {
      this.batchedInvalidations.add(JSON.stringify(key));
    });

    // Clear existing timeout
    if (this.batchTimeout) {
      clearTimeout(this.batchTimeout);
    }

    // Set new timeout
    this.batchTimeout = setTimeout(() => {
      this.flushBatch(options);
    }, 100); // 100ms batch window
  }

  /**
   * Flush batched invalidations
   */
  private async flushBatch(options: Required<InvalidationOptions>): Promise<void> {
    const keysToInvalidate = Array.from(this.batchedInvalidations).map((keyString) =>
      JSON.parse(keyString)
    );

    this.batchedInvalidations.clear();
    this.batchTimeout = null;

    if (keysToInvalidate.length > 0) {
      await this.immediateInvalidate(keysToInvalidate, options);
    }
  }

  /**
   * Deferred invalidation strategy (next tick)
   */
  private deferInvalidate(
    queryKeys: readonly unknown[][],
    options: Required<InvalidationOptions>
  ): void {
    Promise.resolve().then(() => {
      this.immediateInvalidate(queryKeys, options);
    });
  }

  /**
   * Log invalidation for debugging
   */
  private logInvalidation(
    mutationType: keyof typeof InvalidationRegistry,
    context: InvalidationContext,
    queryKeys: readonly unknown[][]
  ): void {
    console.group(`🔄 Cache Invalidation: ${mutationType}`);
    queryKeys.forEach((key, index) => {
    });
    console.groupEnd();
  }

  /**
   * Record invalidation for monitoring
   */
  private recordInvalidation(result: InvalidationResult): void {
    this.invalidationHistory.push(result);

    // Keep only last 100 invalidations
    if (this.invalidationHistory.length > 100) {
      this.invalidationHistory = this.invalidationHistory.slice(-100);
    }
  }

  /**
   * Get invalidation statistics
   */
  getStats(): {
    totalInvalidations: number;
    averageDuration: number;
    successRate: number;
    recentFailures: InvalidationResult[];
  } {
    const total = this.invalidationHistory.length;
    const successful = this.invalidationHistory.filter((r) => r.success).length;
    const averageDuration = total > 0
      ? this.invalidationHistory.reduce((sum, r) => sum + r.duration, 0) / total
      : 0;
    const recentFailures = this.invalidationHistory
      .filter((r) => !r.success)
      .slice(-10);

    return {
      totalInvalidations: total,
      averageDuration,
      successRate: total > 0 ? successful / total : 1,
      recentFailures,
    };
  }

  /**
   * Clear invalidation history
   */
  clearHistory(): void {
    this.invalidationHistory = [];
  }

  /**
   * Invalidate all queries for an entity
   */
  async invalidateEntity(
    entity: string,
    options: InvalidationOptions = {}
  ): Promise<void> {
    const finalOptions = { ...DEFAULT_OPTIONS, ...options };

    await this.queryClient.invalidateQueries({
      predicate: (query) => {
        const queryKey = query.queryKey;
        return Array.isArray(queryKey) && queryKey[0] === entity;
      },
      refetchType: finalOptions.refetchInactive ? 'all' : 'active',
    });

    if (finalOptions.debug) {
      // debug logging removed
    }
  }

  /**
   * Invalidate queries matching a predicate
   */
  async invalidateByPredicate(
    predicate: (queryKey: readonly unknown[]) => boolean,
    options: InvalidationOptions = {}
  ): Promise<void> {
    const finalOptions = { ...DEFAULT_OPTIONS, ...options };

    await this.queryClient.invalidateQueries({
      predicate: (query) => predicate(query.queryKey),
      refetchType: finalOptions.refetchInactive ? 'all' : 'active',
    });

    if (finalOptions.debug) {
      // debug logging removed
    }
  }
}

/**
 * Global invalidation service instance
 */
let invalidationServiceInstance: InvalidationService | null = null;

/**
 * Initialize the invalidation service
 */
export function initInvalidationService(queryClient: QueryClient): InvalidationService {
  invalidationServiceInstance = new InvalidationService(queryClient);
  return invalidationServiceInstance;
}

/**
 * Get the global invalidation service instance
 */
export function getInvalidationService(): InvalidationService {
  if (!invalidationServiceInstance) {
    throw new Error(
      'InvalidationService not initialized. Call initInvalidationService first.'
    );
  }
  return invalidationServiceInstance;
}

/**
 * Convenience function for immediate invalidation
 */
export async function invalidateCache(
  mutationType: keyof typeof InvalidationRegistry,
  context: InvalidationContext = {},
  options: InvalidationOptions = {}
): Promise<InvalidationResult> {
  const service = getInvalidationService();
  return service.invalidate(mutationType, context, options);
}