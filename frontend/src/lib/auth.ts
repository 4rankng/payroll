export type AppRole = 'admin' | 'partner' | 'employee' | 'adv_partner' | 'accountant';

// PAYR-37B item 4: the JWT no longer lives in localStorage. The server sets it
// as an HttpOnly cookie (SameSite=Lax, Secure in production) and client JS
// cannot read it. What remains client-side is the user PROFILE (display data,
// not a credential) and an expiry timestamp for proactive session-expiry UI.
//
// Legacy key kept ONLY for the migration window: sessions created by the old
// bundle still have this token stored; the bootstrap flow exchanges it for a
// cookie once, then deletes it (see bootstrapLegacySession in auth.service).
export const LEGACY_TOKEN_KEY = 'auth_token';

const SESSION_EXPIRES_KEY = 'auth_session_expires_at'; // epoch ms — metadata, not the token
export const USER_ID_KEY = 'userId';

/**
 * Parse an expiry metadata value defensively. Any non-finite/past value means
 * "unknown or expired" depending on the caller.
 */
function readSessionExpiresAt(): number | null {
  try {
    const raw = localStorage.getItem(SESSION_EXPIRES_KEY);
    if (!raw) return null;
    const value = Number(raw);
    return Number.isFinite(value) ? value : null;
  } catch {
    return null;
  }
}

class AuthManager {
  private static instance: AuthManager;
  private tokenExpiryWarningShown = false;
  private sessionTimeoutId: NodeJS.Timeout | null = null;

  private constructor() {
    // Don't start session monitoring automatically
    // It will be started after successful login
  }

  static getInstance(): AuthManager {
    if (!AuthManager.instance) {
      AuthManager.instance = new AuthManager();
    }
    return AuthManager.instance;
  }

  /**
   * Record session expiry metadata after a successful login. The cookie
   * lifetime is managed entirely by the server; this timestamp only drives
   * the client-side "phiên sắp hết hạn" warning and auto-logout timer.
   * Pass null when the remaining lifetime is unknown (legacy bootstrap).
   */
  setSessionMeta(expiresInSeconds: number | null): void {
    try {
      if (expiresInSeconds && expiresInSeconds > 0) {
        localStorage.setItem(SESSION_EXPIRES_KEY, String(Date.now() + expiresInSeconds * 1000));
      } else {
        localStorage.removeItem(SESSION_EXPIRES_KEY);
      }
    } catch {
      // localStorage unavailable (private mode): the session still works —
      // only the proactive expiry warning is lost.
    }
    // Start session monitoring after setting session metadata
    this.startSessionMonitoring();
    this.notifyAuthChanged();
  }

  /**
   * Whether a session is considered active. This is a UI heuristic only —
   * the server is the source of truth (it authenticates every request via
   * the HttpOnly cookie, and a 401 interceptor clears the session).
   */
  hasSession(): boolean {
    if (typeof window === 'undefined') return false;
    let profilePresent = false;
    try {
      profilePresent = localStorage.getItem('userRole') !== null;
    } catch {
      return false;
    }
    if (!profilePresent) return false;
    const expiresAt = readSessionExpiresAt();
    // No expiry metadata (e.g. legacy-bootstrapped session): trust the
    // profile; the server rejects expired sessions on the next request.
    if (expiresAt === null) return true;
    return expiresAt > Date.now();
  }

  getUserId(): number | null {
    try {
      const raw = localStorage.getItem(USER_ID_KEY);
      if (!raw) return null;
      const id = Number(raw);
      return Number.isFinite(id) && id > 0 ? id : null;
    } catch {
      return null;
    }
  }

  clearSession(): void {
    this.clearStoredUser();
    if (this.sessionTimeoutId) {
      clearTimeout(this.sessionTimeoutId);
      this.sessionTimeoutId = null;
    }
    this.notifyAuthChanged();
  }

  startSessionMonitoring(): void {
    if (this.sessionTimeoutId) {
      clearTimeout(this.sessionTimeoutId);
      this.sessionTimeoutId = null;
    }

    const checkSessionExpiry = () => {
      const expiresAt = readSessionExpiresAt();
      if (expiresAt === null) return; // unknown lifetime — server enforces

      const timeUntilExpiry = expiresAt - Date.now();

      // Warn 5 minutes before expiry
      if (timeUntilExpiry <= 5 * 60 * 1000 && timeUntilExpiry > 0 && !this.tokenExpiryWarningShown) {
        this.tokenExpiryWarningShown = true;
        this.showExpiryWarning();
      }

      // Auto logout when the session expires
      if (timeUntilExpiry <= 0) {
        this.handleSessionExpired();
      } else {
        // Check again in 1 minute
        this.sessionTimeoutId = setTimeout(checkSessionExpiry, 60 * 1000);
      }
    };

    checkSessionExpiry();
  }

  private showExpiryWarning(): void {
    // This will be connected to the toast notification system
    const event = new CustomEvent('token-expiry-warning', {
      detail: { message: 'Phiên làm việc sắp hết hạn. Vui lòng đăng nhập lại.' }
    });
    window.dispatchEvent(event);
  }

  private handleSessionExpired(): void {
    this.clearSession();
    const event = new CustomEvent('token-expired', {
      detail: { message: 'Phiên làm việc đã hết hạn. Vui lòng đăng nhập lại.' }
    });
    window.dispatchEvent(event);

    // Only redirect if not already on login page to prevent infinite loops
    if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
      window.location.href = '/login';
    }
  }

  // Check if user has required role
  hasRole(requiredRole: AppRole): boolean {
    const userRole = this.getUserRole();
    return userRole === requiredRole;
  }

  // Check if user has unknown of the required roles
  hasAnyRole(roles: AppRole[]): boolean {
    const userRole = this.getUserRole();
    return userRole !== null && roles.includes(userRole);
  }

  getUserRole(): AppRole | null {
    try {
      return (localStorage.getItem('userRole') as AppRole | null) || null;
    } catch {
      return null;
    }
  }

  /** Legacy token left by the pre-cookie bundle, if any. */
  getLegacyToken(): string | null {
    try {
      return localStorage.getItem(LEGACY_TOKEN_KEY);
    } catch {
      return null;
    }
  }

  /** Remove the legacy token after it has been exchanged for a cookie. */
  clearLegacyToken(): void {
    try {
      localStorage.removeItem(LEGACY_TOKEN_KEY);
    } catch {
      // ignore
    }
  }

  private clearStoredUser(): void {
    try {
      localStorage.removeItem('userRole');
      localStorage.removeItem('userName');
      localStorage.removeItem('userEmail');
      localStorage.removeItem('userUsername');
      localStorage.removeItem(USER_ID_KEY);
      localStorage.removeItem('userStatus');
      localStorage.removeItem('userCreatedAt');
      localStorage.removeItem('userUpdatedAt');
      localStorage.removeItem('userLastLogin');
      localStorage.removeItem(SESSION_EXPIRES_KEY);
      localStorage.removeItem(LEGACY_TOKEN_KEY);
    } catch {
      // ignore
    }
  }

  private notifyAuthChanged(): void {
    if (typeof window === 'undefined') {
      return;
    }

    window.dispatchEvent(new CustomEvent('auth-changed'));

    if ('BroadcastChannel' in window) {
      const channel = new BroadcastChannel('auth-channel');
      channel.postMessage({ type: 'auth-changed' });
      channel.close();
    }
  }
}

export const authManager = AuthManager.getInstance();

// Rate limiting tracker
class RateLimiter {
  private attempts: Map<string, { count: number; resetTime: number }> = new Map();
  private readonly maxAttempts = 5;
  private readonly windowMs = 15 * 60 * 1000; // 15 minutes

  canAttempt(identifier: string): { allowed: boolean; remainingAttempts: number; resetTime?: Date } {
    const now = Date.now();
    const record = this.attempts.get(identifier);

    if (!record || now > record.resetTime) {
      this.attempts.set(identifier, {
        count: 1,
        resetTime: now + this.windowMs
      });
      return { allowed: true, remainingAttempts: this.maxAttempts - 1 };
    }

    if (record.count >= this.maxAttempts) {
      return {
        allowed: false,
        remainingAttempts: 0,
        resetTime: new Date(record.resetTime)
      };
    }

    record.count++;
    return {
      allowed: true,
      remainingAttempts: this.maxAttempts - record.count
    };
  }

  reset(identifier: string): void {
    this.attempts.delete(identifier);
  }
}

export const rateLimiter = new RateLimiter();
