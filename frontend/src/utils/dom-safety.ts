/**
 * DOM Safety Utilities
 * Prevents common DOM manipulation errors that occur with browser extensions
 * and cached content conflicts in production
 */

/**
 * Safely removes a child node from its parent
 * Prevents "removeChild is not a child of this node" errors
 */
export function safeRemoveChild(parent: Node | null, child: Node | null): boolean {
  if (!parent || !child) return false;

  try {
    // Verify the child is actually a child of the parent before removing
    if (parent.contains(child)) {
      parent.removeChild(child);
      return true;
    }
  } catch (error) {
    console.warn('Safe removeChild failed:', error);
  }

  return false;
}

/**
 * Safely inserts a node before another node
 * Prevents DOM manipulation errors during component re-renders
 */
export function safeInsertBefore(parent: Node | null, newNode: Node | null, referenceNode: Node | null): boolean {
  if (!parent || !newNode) return false;

  try {
    // If referenceNode is null or not a child, append to the end
    if (!referenceNode || !parent.contains(referenceNode)) {
      parent.appendChild(newNode);
      return true;
    }

    parent.insertBefore(newNode, referenceNode);
    return true;
  } catch (error) {
    console.warn('Safe insertBefore failed:', error);
  }

  return false;
}

/**
 * Safely appends a child node
 * Handles cases where the node might already be attached elsewhere
 */
export function safeAppendChild(parent: Node | null, child: Node | null): boolean {
  if (!parent || !child) return false;

  try {
    // If child is already attached to another parent, remove it first
    if (child.parentNode && child.parentNode !== parent) {
      safeRemoveChild(child.parentNode, child);
    }

    parent.appendChild(child);
    return true;
  } catch (error) {
    console.warn('Safe appendChild failed:', error);
  }

  return false;
}

/**
 * Clears browser cache-related issues by forcing a reload
 * Use as a last resort when DOM manipulation errors persist
 */
export function clearCacheAndReload(): void {
  try {
    // Clear localStorage (if the app uses it)
    localStorage.clear();

    // Clear sessionStorage (if the app uses it)
    sessionStorage.clear();

    // Force a hard reload to clear any cached resources
    window.location.reload();
  } catch (error) {
    console.warn('Cache clearing failed:', error);
    // Fallback to regular reload
    window.location.reload();
  }
}

/**
 * Detects if running in incognito/private mode
 * Useful for debugging cache-related issues
 */
export function isIncognitoMode(): Promise<boolean> {
  return new Promise((resolve) => {
    try {
      // Try to detect incognito mode using storage quota
      (navigator as { webkitTemporaryStorage?: { queryUsageAndQuota: (cb: (u: number, q: number) => void, err: () => void) => void } }).webkitTemporaryStorage?.queryUsageAndQuota(
        (usage, quota) => {
          // In incognito mode, quota is typically much smaller
          resolve(quota < 120000000); // 120MB threshold
        },
        () => resolve(false)
      );
    } catch {
      // Fallback detection methods
      try {
        const testKey = '_incognito_test_';
        localStorage.setItem(testKey, '1');
        localStorage.removeItem(testKey);
        resolve(false); // Not incognito if localStorage works
      } catch {
        resolve(true); // Likely incognito if localStorage doesn't work
      }
    }
  });
}

/**
 * Creates a safe wrapper for React portals to prevent DOM errors
 */
export function createSafePortalContainer(id: string): HTMLElement | null {
  try {
    let container = document.getElementById(id);

    if (!container) {
      container = document.createElement('div');
      container.id = id;

      // Ensure body exists before appending
      if (document.body) {
        safeAppendChild(document.body, container);
      } else {
        // Wait for body to be available
        const observer = new MutationObserver(() => {
          if (document.body) {
            safeAppendChild(document.body, container);
            observer.disconnect();
          }
        });

        observer.observe(document.documentElement, {
          childList: true,
          subtree: true
        });
      }
    }

    return container;
  } catch (error) {
    console.warn('Failed to create safe portal container:', error);
    return null;
  }
}