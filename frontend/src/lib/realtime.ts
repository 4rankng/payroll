import { authManager } from './auth';

export interface RealtimeEvent {
  type: 'notification' | 'activity' | 'data-update' | 'approval' | 'system';
  action: string;
  payload: unknown;
  timestamp: string;
}

export interface RealtimeConfig {
  url?: string;
  reconnectDelay?: number;
  maxReconnectAttempts?: number;
  heartbeatInterval?: number;
}

class RealtimeService {
  private static instance: RealtimeService;
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private heartbeatTimer: NodeJS.Timeout | null = null;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private eventListeners: Map<string, Set<(event: RealtimeEvent) => void>> = new Map();
  private config: Required<RealtimeConfig> = {
    url: process.env.VITE_WS_URL || 'ws://localhost:3001',
    reconnectDelay: 3000,
    maxReconnectAttempts: 5,
    heartbeatInterval: 30000
  };
  private isConnecting = false;
  private messageQueue: RealtimeEvent[] = [];

  private constructor() {}

  static getInstance(): RealtimeService {
    if (!RealtimeService.instance) {
      RealtimeService.instance = new RealtimeService();
    }
    return RealtimeService.instance;
  }

  configure(config: RealtimeConfig): void {
    this.config = { ...this.config, ...config };
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN || this.isConnecting) {
      return;
    }

    const token = authManager.getToken();
    if (!token) {
      console.warn('No auth token available for WebSocket connection');
      return;
    }

    this.isConnecting = true;

    try {
      // Include token in WebSocket URL or use subprotocol
      this.ws = new WebSocket(`${this.config.url}?token=${token}`);

      this.ws.onopen = () => {
        this.isConnecting = false;
        this.reconnectAttempts = 0;
        this.startHeartbeat();
        this.flushMessageQueue();
        this.emit('system', 'connected', { timestamp: new Date().toISOString() });
      };

      this.ws.onmessage = (event) => {
        try {
          const data: RealtimeEvent = JSON.parse(event.data);
          this.handleMessage(data);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        this.isConnecting = false;
        this.emit('system', 'error', { error: 'Connection error' });
      };

      this.ws.onclose = () => {
        this.isConnecting = false;
        this.stopHeartbeat();
        this.emit('system', 'disconnected', { timestamp: new Date().toISOString() });
        this.attemptReconnect();
      };
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      this.isConnecting = false;
      this.attemptReconnect();
    }
  }

  disconnect(): void {
    this.stopHeartbeat();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      this.emit('system', 'reconnect-failed', {
        attempts: this.reconnectAttempts,
        maxAttempts: this.config.maxReconnectAttempts
      });
      return;
    }

    this.reconnectAttempts++;
    const delay = this.config.reconnectDelay * Math.min(this.reconnectAttempts, 3);


    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, delay);
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();
    this.heartbeatTimer = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.send('system', 'heartbeat', { timestamp: new Date().toISOString() });
      }
    }, this.config.heartbeatInterval);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private handleMessage(event: RealtimeEvent): void {
    const listeners = this.eventListeners.get(event.type);
    if (listeners) {
      listeners.forEach(listener => {
        try {
          listener(event);
        } catch (error) {
          console.error('Error in event listener:', error);
        }
      });
    }

    // Also emit to specific action listeners
    const actionListeners = this.eventListeners.get(`${event.type}:${event.action}`);
    if (actionListeners) {
      actionListeners.forEach(listener => {
        try {
          listener(event);
        } catch (error) {
          console.error('Error in action listener:', error);
        }
      });
    }
  }

  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0) {
      const event = this.messageQueue.shift();
      if (event) {
        this.sendRaw(event);
      }
    }
  }

  send(type: string, action: string, payload: unknown): void {
    const event: RealtimeEvent = {
      type: type as RealtimeEvent['type'],
      action,
      payload,
      timestamp: new Date().toISOString()
    };

    if (this.ws?.readyState === WebSocket.OPEN) {
      this.sendRaw(event);
    } else {
      this.messageQueue.push(event);
      if (!this.isConnecting) {
        this.connect();
      }
    }
  }

  private sendRaw(event: RealtimeEvent): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(event));
    }
  }

  emit(type: string, action: string, payload: unknown): void {
    const event: RealtimeEvent = {
      type: type as RealtimeEvent['type'],
      action,
      payload,
      timestamp: new Date().toISOString()
    };
    this.handleMessage(event);
  }

  on(eventType: string, listener: (event: RealtimeEvent) => void): () => void {
    if (!this.eventListeners.has(eventType)) {
      this.eventListeners.set(eventType, new Set());
    }

    const listeners = this.eventListeners.get(eventType)!;
    listeners.add(listener);

    // Return unsubscribe function
    return () => {
      listeners.delete(listener);
      if (listeners.size === 0) {
        this.eventListeners.delete(eventType);
      }
    };
  }

  once(eventType: string, listener: (event: RealtimeEvent) => void): () => void {
    const wrapper = (event: RealtimeEvent) => {
      listener(event);
      unsubscribe();
    };
    const unsubscribe = this.on(eventType, wrapper);
    return unsubscribe;
  }

  getConnectionState(): 'connecting' | 'connected' | 'disconnected' {
    if (this.isConnecting) return 'connecting';
    if (this.ws?.readyState === WebSocket.OPEN) return 'connected';
    return 'disconnected';
  }
}

export const realtimeService = RealtimeService.getInstance();

// React hooks for realtime service
import { useEffect, useState, useCallback } from 'react';

export function useRealtimeConnection() {
  const [connectionState, setConnectionState] = useState<'connecting' | 'connected' | 'disconnected'>('disconnected');

  useEffect(() => {
    realtimeService.connect();

    const unsubscribeConnected = realtimeService.on('system:connected', () => {
      setConnectionState('connected');
    });

    const unsubscribeDisconnected = realtimeService.on('system:disconnected', () => {
      setConnectionState('disconnected');
    });

    const checkState = setInterval(() => {
      setConnectionState(realtimeService.getConnectionState());
    }, 1000);

    return () => {
      unsubscribeConnected();
      unsubscribeDisconnected();
      clearInterval(checkState);
    };
  }, []);

  return connectionState;
}

export function useRealtimeEvent<T = unknown>(
  eventType: string,
  handler: (payload: T) => void
) {
  useEffect(() => {
    const unsubscribe = realtimeService.on(eventType, (event) => {
      handler(event.payload as T);
    });

    return unsubscribe;
  }, [eventType, handler]);
}

export function useRealtimeSend() {
  return useCallback((type: string, action: string, payload: unknown) => {
    realtimeService.send(type, action, payload);
  }, []);
}
