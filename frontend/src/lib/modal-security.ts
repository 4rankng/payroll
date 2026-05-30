import CryptoJS from 'crypto-js';
import { authManager } from '@/lib/auth';

/**
 * Modal Security Utilities
 * Handles encryption, hashing, and secure parameter management for modal deep links
 */

// Get session-based encryption key
const getEncryptionKey = (): string => {
  const token = authManager.getToken();
  if (!token) throw new Error('No active session for encryption');
  
  // Use first 32 chars of token as encryption key
  return CryptoJS.SHA256(token).toString().substring(0, 32);
};

/**
 * Encrypt sensitive data for URL parameters
 */
export const encryptModalData = (data: Record<string, unknown>): string => {
  try {
    const key = getEncryptionKey();
    const encrypted = CryptoJS.AES.encrypt(JSON.stringify(data), key).toString();
    // URL-safe base64 encoding
    return encrypted.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
  } catch (error) {
    console.error('Failed to encrypt modal data:', error);
    throw new Error('Encryption failed');
  }
};

/**
 * Decrypt sensitive data from URL parameters
 */
export const decryptModalData = <T = Record<string, unknown>>(encryptedData: string): T => {
  try {
    const key = getEncryptionKey();
    // Restore base64 padding and characters
    const restored = encryptedData.replace(/-/g, '+').replace(/_/g, '/');
    const padded = restored + '==='.slice((restored.length + 3) % 4);
    
    const decrypted = CryptoJS.AES.decrypt(padded, key);
    const decryptedString = decrypted.toString(CryptoJS.enc.Utf8);
    
    if (!decryptedString) {
      throw new Error('Decryption failed - invalid data or key');
    }
    
    return JSON.parse(decryptedString) as T;
  } catch (error) {
    console.error('Failed to decrypt modal data:', error);
    throw new Error('Decryption failed');
  }
};

/**
 * Generate secure hash for IDs to prevent enumeration
 */
export const hashId = (id: number | string): string => {
  const key = getEncryptionKey();
  const hash = CryptoJS.HmacSHA256(id.toString(), key).toString();
  // Return first 12 characters for shorter URLs
  return hash.substring(0, 12);
};

/**
 * Verify hashed ID matches original
 */
export const verifyHashedId = (id: number | string, hash: string): boolean => {
  try {
    return hashId(id) === hash;
  } catch {
    return false;
  }
};

/**
 * Sanitize URL parameters to prevent XSS
 */
export const sanitizeParam = (param: string): string => {
  return param
    .replace(/[<>]/g, '') // Remove potential HTML tags
    .replace(/javascript:/gi, '') // Remove javascript: protocol
    .replace(/data:/gi, '') // Remove data: protocol
    .replace(/vbscript:/gi, '') // Remove vbscript: protocol
    .trim();
};

/**
 * Validate parameter format
 */
export const isValidParam = (param: string, type: 'hash' | 'encrypted' | 'plain' = 'plain'): boolean => {
  if (!param || param.length === 0) return false;
  
  switch (type) {
    case 'hash':
      // Hash should be 12 alphanumeric characters
      return /^[a-f0-9]{12}$/.test(param);
    case 'encrypted':
      // Encrypted data should be URL-safe base64
      return /^[A-Za-z0-9\-_]+$/.test(param);
    case 'plain':
      // Allow alphanumeric, dash, underscore
      return /^[a-zA-Z0-9\-_]+$/.test(param);
    default:
      return false;
  }
};

/**
 * Security context for session validation
 */
export interface SecurityContext {
  userId: number;
  userRole: string;
  sessionId: string;
  permissions: string[];
  timestamp: number;
}

/**
 * Get permissions based on user role
 */
const getRoleBasedPermissions = (role: string): string[] => {
  if (role === 'admin') {
    return [
      'timesheet:approve',
      'timesheet:reject',
      'timesheet:edit',
      'timesheet:delete',
      'timesheet:create',
      'employee:create',
      'employee:edit',
      'employee:delete',
      'project:create',
      'project:edit',
      'project:delete',
      'payrate:create',
      'payrate:edit',
      'payrate:delete',
      'payrate:approve',
      'payroll:manage',
      'user:manage',
      'reports:view'
    ];
  }
  
  if (role === 'partner') {
    return [
      'timesheet:edit',
      'timesheet:create',
      'employee:create',
      'employee:edit',
      'project:create',
      'project:edit',
      'payrate:create',
      'payrate:edit',
      'reports:view',
      'user:manage' // Allow partners to manage their own user account (change password, update profile)
    ];
  }
  
  return [];
};

/**
 * Get current security context
 */
export const getSecurityContext = (): SecurityContext => {
  const token = authManager.getToken();
  const role = authManager.getUserRole();
  const userId = authManager.getUserId();
  
  if (!token || !role || !userId) {
    throw new Error('Invalid session - user not authenticated');
  }
  
  return {
    userId,
    userRole: role,
    sessionId: CryptoJS.SHA256(token).toString().substring(0, 16),
    permissions: getRoleBasedPermissions(role),
    timestamp: Date.now()
  };
};

/**
 * Validate security context for modal access
 */
export const validateSecurityContext = (context: SecurityContext): boolean => {
  try {
    const current = getSecurityContext();
    
    // Validate session hasn't changed
    if (context.sessionId !== current.sessionId) {
      return false;
    }
    
    // Validate user hasn't changed
    if (context.userId !== current.userId || context.userRole !== current.userRole) {
      return false;
    }
    
    // Validate timestamp (context valid for 1 hour)
    const oneHour = 60 * 60 * 1000;
    if (Date.now() - context.timestamp > oneHour) {
      return false;
    }
    
    return true;
  } catch {
    return false;
  }
};