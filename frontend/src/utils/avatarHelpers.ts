/**
 * Neutral profile glyph used wherever an account has no uploaded photo.
 *
 * A soft emerald disc with a simple human-abstract silhouette. It is neither a
 * real portrait, a stylised character, nor initials, so identity visuals stay
 * occupation- and appearance-neutral across admin, partner and employee
 * surfaces. The glyph is intentionally identical for every account — the name
 * rendered beside it carries identity.
 *
 * Drawn inline as an SVG data URI so it renders crisply at any size, keeps the
 * theme palette, and needs no network request or image asset.
 */
const NEUTRAL_AVATAR_SVG =
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">' +
  '<rect width="100" height="100" fill="#eaf8f0"/>' +
  '<circle cx="50" cy="37" r="15" fill="#08783e"/>' +
  '<path d="M50 57c-15.5 0-28 9.4-28 21v6h56v-6c0-11.6-12.5-21-28-21z" fill="#08783e"/>' +
  "</svg>";

// Percent-encode the SVG: raw `#` in a data URI starts a fragment identifier and
// truncates the document, and `<`/`>` are not valid in a URI path.
const NEUTRAL_AVATAR_DATA_URL = `data:image/svg+xml,${encodeURIComponent(NEUTRAL_AVATAR_SVG)}`;

/**
 * Returns the avatar image source for an account.
 *
 * The signature keeps its historical shape so existing call sites (auth
 * payload builders, UserAvatar) need no change, but the result is the same
 * neutral glyph for every seed by design.
 */
export function generateAvatarUrl(): string {
  return NEUTRAL_AVATAR_DATA_URL;
}
