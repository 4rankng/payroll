import { jwtDecode } from 'jwt-decode';

export type AppRole = 'admin' | 'partner' | 'employee' | 'adv_partner' | 'accountant';

interface TokenPayload {
  exp: number;
  user_id: number;
  username: string;
  role: AppRole;
  sub: string;
  nbf: number;
  iat: number;
  jti: string;
}

class AuthManager {
  private static instance: AuthManager;
  private tokenKey = 'auth_token';
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

  // Use localStorage for persistent 7-day login
  // sessionStorage clears on tab close — users would need to re-login every time
  setToken(token: string): void {
    localStorage.setItem(this.tokenKey, token);
    // Start session monitoring after setting token
    this.startSessionMonitoring();
    this.notifyAuthChanged();
  }

  getToken(): string | null {
    const token = localStorage.getItem(this.tokenKey);

    return token;
  }


  removeToken(): void {
    localStorage.removeItem(this.tokenKey);
    this.clearStoredUser();
    if (this.sessionTimeoutId) {
      clearTimeout(this.sessionTimeoutId);
      this.sessionTimeoutId = null;
    }
    this.notifyAuthChanged();
  }

  isTokenValid(): boolean {
    const token = this.getToken();
    if (!token) return false;

    try {
      const decoded = jwtDecode<TokenPayload>(token);
      const currentTime = Date.now() / 1000;
      return decoded.exp > currentTime;
    } catch {
      return false;
    }
  }

  getTokenPayload(): TokenPayload | null {
    const token = this.getToken();
    if (!token) return null;

    try {
      return jwtDecode<TokenPayload>(token);
    } catch {
      return null;
    }
  }

  getUserRole(): AppRole | null {
    const payload = this.getTokenPayload();

    return payload?.role || null;
  }

  getUserId(): number | null {
    const payload = this.getTokenPayload();

    return payload?.user_id || null;
  }

  getTokenExpiry(): Date | null {
    const payload = this.getTokenPayload();
    if (!payload) return null;
    return new Date(payload.exp * 1000);
  }

  startSessionMonitoring(): void {
    if (this.sessionTimeoutId) {
      clearTimeout(this.sessionTimeoutId);
    }

    const checkTokenExpiry = () => {
      const token = this.getToken();
      if (!token) return;

      const expiry = this.getTokenExpiry();
      if (!expiry) return;

      const now = new Date();
      const timeUntilExpiry = expiry.getTime() - now.getTime();

      // Warn 5 minutes before expiry
      if (timeUntilExpiry <= 5 * 60 * 1000 && timeUntilExpiry > 0 && !this.tokenExpiryWarningShown) {
        this.tokenExpiryWarningShown = true;
        this.showExpiryWarning();
      }

      // Auto logout when token expires
      if (timeUntilExpiry <= 0) {
        this.handleTokenExpired();
      } else {
        // Check again in 1 minute
        this.sessionTimeoutId = setTimeout(checkTokenExpiry, 60 * 1000);
      }
    };

    checkTokenExpiry();
  }

  private showExpiryWarning(): void {
    // This will be connected to the toast notification system
    const event = new CustomEvent('token-expiry-warning', {
      detail: { message: 'Phiên làm việc sắp hết hạn. Vui lòng đăng nhập lại.' }
    });
    window.dispatchEvent(event);
  }

  private handleTokenExpired(): void {
    this.removeToken();
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

  private clearStoredUser(): void {
    localStorage.removeItem('userRole');
    localStorage.removeItem('userName');
    localStorage.removeItem('userEmail');
    localStorage.removeItem('userUsername');
    localStorage.removeItem('userStatus');
    localStorage.removeItem('userCreatedAt');
    localStorage.removeItem('userUpdatedAt');
    localStorage.removeItem('userLastLogin');
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
