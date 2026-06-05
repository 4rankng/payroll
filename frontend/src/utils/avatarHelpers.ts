import { createAvatar } from '@dicebear/core';
import * as collection from '@dicebear/collection';

const avatarDataUrlCache = new Map<string, string>();
const initialsCache = new Map<string, string>();

export function generateAvatarUrl(seed: string): string {
  const cleanSeed = seed || 'default';

  if (avatarDataUrlCache.has(cleanSeed)) {
    return avatarDataUrlCache.get(cleanSeed)!;
  }

  const avatar = createAvatar(collection.lorelei, {
    seed: cleanSeed,
    backgroundColor: ['ffffff'],
    skinColor: ['f0f0f0'],
    hairColor: ['000000'],
    clothingColor: ['000000'],
    accessoriesColor: ['000000'],
    glassesColor: ['000000'],
  } as unknown as Record<string, string[]>);

  const dataUrl = avatar.toDataUri();
  avatarDataUrlCache.set(cleanSeed, dataUrl);

  return dataUrl;
}

export function generateInitials(name?: string): string {
  if (!name) return '?';

  if (initialsCache.has(name)) {
    return initialsCache.get(name)!;
  }

  const initials = name
    .split(' ')
    .map(part => part.charAt(0).toUpperCase())
    .join('')
    .slice(0, 2);

  initialsCache.set(name, initials);

  return initials;
}

export function generateInitialsFromEmail(email?: string): string {
  if (!email) return '?';
  return email.charAt(0).toUpperCase();
}

export function getUserInitials(name?: string): string {
  if (name) return generateInitials(name);
  return '?';
}

export function clearAvatarCache(): void {
  avatarDataUrlCache.clear();
  initialsCache.clear();
}

export function getAvatarCacheStats(): { urlCacheSize: number; initialsCacheSize: number } {
  return {
    urlCacheSize: avatarDataUrlCache.size,
    initialsCacheSize: initialsCache.size,
  };
}
