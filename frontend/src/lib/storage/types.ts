import { z } from 'zod';

// Zod schemas for runtime validation
export const UserActionSchema = z.object({
  id: z.string(),
  type: z.string(),
  label: z.string(),
  icon: z.string(),
  path: z.string().optional(),
  handler: z.string().optional(),
  timestamp: z.date(),
  userId: z.string(),
});

export const UserPreferencesSchema = z.object({
  userId: z.string(),
  pinnedActions: z.array(z.string()),
  tableSettings: z.record(z.string(), z.object({
    columns: z.array(z.string()),
    pageSize: z.number(),
    sortBy: z.string().optional(),
    sortOrder: z.enum(['asc', 'desc']).optional(),
  })),
  recentSearches: z.array(z.string()),
  language: z.enum(['vi', 'en']),
  sidebarCollapsed: z.boolean(),
  updatedAt: z.date(),
});

export const ActionFrequencySchema = z.object({
  actionId: z.string(),
  userId: z.string(),
  count: z.number(),
  lastUsed: z.date(),
});

// TypeScript types inferred from Zod schemas
export type UserAction = z.infer<typeof UserActionSchema>;
export type UserPreferences = z.infer<typeof UserPreferencesSchema>;
export type ActionFrequency = z.infer<typeof ActionFrequencySchema>;

// Validation functions
export const validateUserAction = (data: unknown): UserAction => {
  return UserActionSchema.parse(data);
};

export const validateUserPreferences = (data: unknown): UserPreferences => {
  return UserPreferencesSchema.parse(data);
};

export const validateActionFrequency = (data: unknown): ActionFrequency => {
  return ActionFrequencySchema.parse(data);
};

// Safe validation functions that return success/error
export const safeValidateUserAction = (data: unknown) => {
  return UserActionSchema.safeParse(data);
};

export const safeValidateUserPreferences = (data: unknown) => {
  return UserPreferencesSchema.safeParse(data);
};

export const safeValidateActionFrequency = (data: unknown) => {
  return ActionFrequencySchema.safeParse(data);
};