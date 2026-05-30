import Dexie, { type Table } from 'dexie';
import { UserAction, UserPreferences, ActionFrequency } from './types';

export class AppDatabase extends Dexie {
  userActions!: Table<UserAction>;
  userPreferences!: Table<UserPreferences>;
  actionFrequency!: Table<ActionFrequency>;

  constructor() {
    super('PayrollAppStorage');
    
    this.version(1).stores({
      userActions: 'id, userId, timestamp, type',
      userPreferences: 'userId',
      actionFrequency: '[actionId+userId], userId, count'
    });
  }
}

// Create database instance
export const db = new AppDatabase();

export class IndexedDBManager {
  async initialize(): Promise<void> {
    try {
      await db.open();
    } catch (error) {
      console.error('Failed to initialize database:', error);
      throw new Error('Failed to open IndexedDB');
    }
  }

  // User Actions
  async addUserAction(action: UserAction): Promise<void> {
    await db.userActions.add(action);
  }

  async getUserActions(userId: string, limit: number = 100): Promise<UserAction[]> {
    return db.userActions
      .where('userId')
      .equals(userId)
      .reverse()
      .limit(limit)
      .toArray();
  }

  async getUserActionsByType(userId: string, type: string): Promise<UserAction[]> {
    return db.userActions
      .where(['userId', 'type'])
      .equals([userId, type])
      .reverse()
      .toArray();
  }

  async clearOldUserActions(userId: string, olderThanDays: number = 30): Promise<void> {
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - olderThanDays);
    
    await db.userActions
      .where('userId')
      .equals(userId)
      .and(action => action.timestamp < cutoffDate)
      .delete();
  }

  // User Preferences
  async getUserPreferences(userId: string): Promise<UserPreferences | undefined> {
    return db.userPreferences.get(userId);
  }

  async setUserPreferences(preferences: UserPreferences): Promise<void> {
    await db.userPreferences.put(preferences);
  }

  async updateUserPreferences(userId: string, updates: Partial<Omit<UserPreferences, 'userId'>>): Promise<void> {
    await db.userPreferences.update(userId, updates);
  }

  // Action Frequency
  async getActionFrequency(userId: string): Promise<ActionFrequency[]> {
    return db.actionFrequency
      .where('userId')
      .equals(userId)
      .reverse()
      .toArray();
  }

  async incrementActionFrequency(actionId: string, userId: string): Promise<void> {
    const existing = await db.actionFrequency.get([actionId, userId]);
    
    if (existing) {
      await db.actionFrequency.update([actionId, userId], { 
        count: existing.count + 1,
        lastUsed: new Date()
      });
    } else {
      await db.actionFrequency.add({
        actionId,
        userId,
        count: 1,
        lastUsed: new Date()
      });
    }
  }

  async getTopActions(userId: string, limit: number = 10): Promise<ActionFrequency[]> {
    const allFrequencies = await db.actionFrequency
      .where('userId')
      .equals(userId)
      .toArray();
    
    return allFrequencies
      .sort((a, b) => b.count - a.count)
      .slice(0, limit);
  }

  async clearUserData(userId: string): Promise<void> {
    await db.transaction('rw', db.userActions, db.userPreferences, db.actionFrequency, async () => {
      await db.userActions.where('userId').equals(userId).delete();
      await db.userPreferences.delete(userId);
      await db.actionFrequency.where('userId').equals(userId).delete();
    });
  }
}

export const indexedDBManager = new IndexedDBManager();