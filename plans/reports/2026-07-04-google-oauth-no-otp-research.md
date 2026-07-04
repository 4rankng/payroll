# Research: Is "Google OAuth, no app OTP" safe for a payroll SaaS?

**Date:** 2026-07-04 · **Scope:** Threat model + industry norms + residual risk + recommendations
**Stated assumption:** admin/partner users "use Google OAuth responsibly" — i.e., Google 2-Step Verification (2SV) is on.
**Decision feeding:** ship "Google login → no OTP" (already implemented in commit `e5dd9f7`) as-is, or add step-up auth.

---

## Executive Summary

**The honest answer: under the stated assumption, "Google OAuth, no OTP" is *defensible* but rests on an assumption the app cannot verify or enforce.** The single most important finding from this research is that **Google's id_token carries no claim about the user's 2SV status, the factor used, or device attestation**, and Google's RISC/Cross-Account Protection API exposes *security-state-change events* (session revoked, credential changed) — **not** a queryable "is 2SV on?" attribute. So "assume admins use 2SV responsibly" is a *policy* the app trusts blindly, with no technical lever to confirm or revoke trust when it's violated.

Two further findings sharpen the picture:

1. **2SV is irrelevant against the dominant 2024–2025 attack: session-cookie theft by infostealers (Lumma, RedLine).** Stolen Google session cookies bypass 2SV entirely because they represent an *already-authenticated* session. Several malware families exploit undocumented Google OAuth2 token-refresh to **regenerate cookies even after a password reset**, making access persistent. So "2SV on" protects the *login step* but not the *post-login session* — and your app mints a 14-day JWT on a successful Google login.

2. **Money-moving B2B SaaS (Stripe, Coinbase, Binance) do NOT trust Google/OAuth alone.** They require app-level MFA or step-up auth on money actions regardless of login method. The "step-up authentication" pattern (SSO for convenience at the door, app MFA for verification at the vault) is the industry norm for this risk class.

**Recommendation:** Ship "Google login → no OTP" **only with compensating controls** (priority-ordered below). The single highest-leverage one is **step-up OTP on the money-moving routes** — it's the control that defends against *both* session-cookie theft *and* a broken 2SV assumption, because it requires fresh verification at the moment of harm.

---

## 1. Threat model when trusting Google OAuth alone

### What a verified Google id_token guarantees
- ✅ Signature (Google signed it)
- ✅ Audience (`aud == your GOOGLE_CLIENT_ID`)
- ✅ Issuer (`iss == accounts.google.com`)
- ✅ Expiry, issue time
- ✅ `email_verified` (Google confirmed the email belongs to the account)

### What the id_token does NOT guarantee
- ❌ **The user's 2SV status** — no claim says "this account has 2SV on"
- ❌ **Which factor was used** to authenticate this session — could be a password, a passkey, a remembered device, or a stolen cookie
- ❌ **Device attestation / phishing-resistance** — Google issues no "this session used a FIDO2 key" signal in the id_token
- ❌ **That the session wasn't established via a stolen cookie** — the id_token is minted for any valid Google session, however that session arose

This is the core problem. The stated assumption ("2SV is on") is invisible to the relying party.

### Attack vectors against a Google-2SV-on user

| Vector | Does Google 2SV mitigate? | Notes |
|--------|:---:|---|
| **Phished Google password** (attacker enters password at Google) | ✅ If 2SV factor is phishing-*resistant* (security key / passkey); ❌ if SMS/TOTP (phishable, relayable) | CISA's 2025 position: only phishing-resistant factors count. SMS is SIM-swappable; TOTP is relayable via AitM proxies (Evilginx). |
| **Session-cookie theft (infostealer: Lumma, RedLine, Stealc)** | ❌ **No** | The dominant 2024-2025 vector. Cookies represent an already-authenticated session; 2SV already happened. [Recorded Future], [Malwarebytes Jan 2024], [Huntress]. |
| **Persistent cookie regeneration** (malware exploits undocumented OAuth2 refresh) | ❌ **No** | Access survives password reset. [InfoStealers.com], [Malwarebytes]. |
| **OAuth malicious-app consent phishing** | ❌ **No** | Attacker's app gets a refresh token; never touches 2SV. |
| **Browser-sync / profile-theft** (signing in to Chrome on a shared device syncs the Google session) | ❌ **No** | Bypasses 2SV entirely. |
| **Token relay / AitM (Evilginx)** on the *Google* login | ✅ Only with phishing-resistant factors (FIDO2 origin-binding); ❌ with SMS/TOTP | [Yubico — phishing-resistant MFA]. |

**Bottom line:** "2SV on" meaningfully defends only the *initial Google login*, and only if the 2SV factor is phishing-*resistant* (security key / passkey). It does **nothing** against cookie/session theft — which is the dominant attack for accounts that have already authenticated. Since your app mints a 14-day JWT on Google login, a stolen Google session cookie → a stolen 14-day payroll-admin token.

---

## 2. Industry norms for money-moving / privileged B2B SaaS

### What the major players do

| Platform | Trust SSO alone for admins? | Layer app-level MFA? |
|----------|:---:|---|
| **Coinbase** | Doesn't offer "Sign in with Google" as sole factor | ✅ Mandatory TOTP/security key on every account — [Coinbase 2SV] |
| **Binance / Binance.US** | No | ✅ At least one 2FA method mandatory — [Binance.US 2FA] |
| **Stripe** | SAML SSO for teams | ✅ Native 2-step auth **or** IdP-enforced MFA — [Stripe 2-step] |
| **AWS** | IAM Identity Center (SSO) | ✅ MFA enforced at the IdP — [Reddit/AWS discussion] |
| **Cloudflare Access** | SSO (any IdP) | ✅ IdP-based MFA **or** Cloudflare-hosted MFA — [Cloudflare MFA] |

**The pattern is consistent:** money-moving platforms treat SSO as a *convenience*, not a *security boundary*. They layer app-level MFA — either always (Coinbase/Binance) or at sensitive actions (Stripe/Cloudflare step-up).

### The "step-up authentication" pattern
- Login via SSO/Google → no extra friction (matches what you built)
- Sensitive actions (wire transfer, disbursement approval, password change, data export) → **fresh MFA challenge**, even mid-session
- Defends against: stolen long-lived token, broken 2SV assumption, malicious insider with momentary access
- Documented as the recommended pattern for risk-based auth — [Ping Identity step-up], [Auth0 step-up]

### Regulatory / standards view
- **NIST SP 800-63B/C** (federated identity): a Relying Party MAY accept a trusted IdP's assertion at AAL2 — **but** the RP inherits the IdP's assurance level. If you can't *verify* Google enforced AAL2 (you can't — see §1), your effective assurance is AAL1 (single-factor). For privileged access to financial data, AAL2 is the floor. — [NIST SP 800-63B], [NIST SP 800-63C]
- **SOC 2 / ISO 27001**: auditors expect MFA on privileged accounts; "we trust Google" is a finding unless you can demonstrate enforcement.
- **OWASP ASVS**: requires step-up or re-authentication for high-impact transactions.

**Bottom line:** The industry bar for a money-moving, multi-tenant, admin-privileged app is **not** "trust Google alone." It's "trust Google for login, verify yourself at the vault."

---

## 3. The residual risk under the stated assumption

### Is the assumption realistic to rely on?
**No — because it is unverifiable and unenforceable from the app side.**

- The Google id_token carries **no 2SV-status claim** — confirmed via the [idtoken.Payload] struct fields (`iss`, `aud`, `sub`, `email`, `email_verified`; no `amr`, no `loa`, no `2sv`).
- **Google's RISC / Cross-Account Protection API** delivers *events* (session revoked, credential changed, account disabled) as Security Event Tokens — **not** a queryable "is 2SV enabled?" attribute. [Google RISC docs], [Google Developers Blog].
- The only ways to know 2SV status:
  - **Google Workspace Admin API** — only works if your admins are on a Workspace tenant you manage. For consumer Gmail accounts (likely for `frankng.sg@gmail.com`-style logins), there is **no API**.
  - **Manual attestation** — ask the user to confirm; trust but don't verify.

So "assume admins use Google OAuth responsibly" is a **policy the app trusts blindly**, with no technical lever to confirm or auto-revoke when an admin turns 2SV off (or never had it on a phishing-resistant factor).

### Realistic remaining attack surface (Google-2SV-on user, no app OTP)
1. **Session-cookie theft via infostealer** — the #1 vector. 2SV is irrelevant. Your 14-day JWT becomes the attacker's 14-day payroll-admin token. This is the realistic threat, not a hypothetical.
2. **OAuth malicious-app consent** — attacker's app gets a Google refresh token, never touches 2SV.
3. **2SV factor is weak** — if the admin's "2SV" is SMS or even TOTP, phishing/relay still works. Only FIDO2/passkey is phishing-resistant.
4. **Admin disables 2SV** — you won't know.

### Is "Google OAuth + assumed 2SV" equivalent to "password + email OTP"?
**No — they defend against different things:**
- **Password + email OTP** defends against: *password-only* compromise (leaked `Admin123`, phished password). The attacker needs *both* the password and inbox access.
- **Google OAuth (no app OTP)** defends against: *nothing additional* beyond what Google's own session already provides. A stolen Google session = a stolen app session, instantly. Email OTP would have added an independent second channel the cookie-thief doesn't control.

The two are **not equivalent**. Email OTP on top of Google login would actually close the session-cookie-theft gap (the attacker has the Google cookie but not the inbox). Removing it (your current state) reopens that gap.

---

## 4. Concrete recommendations (priority order)

For a Vietnam-based payroll SaaS, admin/partner roles, JWT 14-day TTL, OnePay disbursements, multi-tenant — given the owner has decided Google-login = no OTP.

### 🥇 #1 — Step-up OTP on money-moving routes (highest leverage)
- Reuse the OTP infrastructure you just built. Add a `RequireStepUpOTP()` middleware on **only** the money routes: `POST /admin/manual-disbursement`, `/wallet/adjust`, bulk transfers, OnePay settlement approval.
- Even if the user logged in via Google 5 days ago, approving a payout re-prompts for an emailed code.
- **Why it's #1:** it defends against *both* session-cookie theft (the thief has the token but not the inbox at the moment of payout) *and* a broken 2SV assumption. It's the Stripe/Cloudflare pattern. Cost: ~half a day, reusing existing infra.

### 🥈 #2 — Shorten the JWT TTL for admin/partner (cheapest)
- 14 days is too long for a session that can move money. Drop admin/partner access TTL to **8–24 hours** (configurable). Employee can stay at 14 days.
- **Why:** bounds the window of a stolen-token/cookie attack. Cheap (one config change + claim).
- Tradeoff: more re-logins. Acceptable for admin/partner (small population, high value).

### 🥉 #3 — Document + policy-enforce the Google 2SV requirement
- Add to onboarding/runbook: *"Every admin/partner Google account MUST have 2-Step Verification enabled, ideally a security key or passkey (not SMS)."*
- If you manage a Google Workspace tenant for the org, **enforce 2SV via Workspace Admin** (enforceable, auditable). This is the one case where the assumption becomes verifiable.
- For consumer Gmail accounts, this is trust-without-verify — accept it as a documented residual risk.

### 🔬 #4 (optional, higher effort) — Subscribe to Google RISC / Cross-Account Protection
- Receive `sessionsRevoked` / `accountCredentialChangeRecover` / `accountDisabled` events and **revoke the user's app JWT** in response (you already have `tokens_invalid_before` from commit `619f9cd`).
- Doesn't close the cookie-theft gap (the thief's session is still valid until Google kills it) but reduces dwell time after Google detects abuse.
- ~1-2 days integration. Worth it if the admin population grows.

### What you should NOT bother with
- ❌ **Trying to verify 2SV status via API** — not exposed for consumer Gmail accounts; only works inside a Workspace tenant you manage.
- ❌ **WebAuthn/FIDO2 as the app's own factor right now** — the right v2 (phishing-resistant, origin-bound, defeats cookie theft entirely), but a meaningful build. Track separately.
- ❌ **Re-adding OTP to the Google *login* step** — you've decided against it; step-up on money actions (#1) is strictly better than blanket login OTP.

---

## Unresolved questions

1. **Are your admin/partner accounts on a Google Workspace tenant you manage, or consumer Gmail?** If Workspace → you can enforce 2SV centrally (recommendation #3 becomes strong). If consumer Gmail → it's a documented trust gap.
2. **What's the realistic admin/partner session frequency?** If they log in daily, an 8h TTL (#2) is painless. If weekly, it's friction.
3. **Is the OnePay disbursement flow approvable by a single admin, or does it require two-person approval?** Two-person approval (maker-checker) is a stronger control than any OTP for money movement and is the banking standard.

---

## Sources

### Threat model / session theft
- [Huntress — From Cookies to Keys: Session Hijacking](https://www.huntress.com/blog/why-hackers-don't-need-passwords-anymore)
- [Recorded Future — Session Hijacking and MFA Bypass](https://www.recordedfuture.com/blog/session-hijacking-mfa-bypass)
- [Malwarebytes (Jan 2024) — Info-Stealers persistent Google account access](https://www.malwarebytes.com/blog/news/2024/01/info-stealers-can-steal-cookies-for-permanent-access-to-your-google-account)
- [InfoStealers.com — OAuth2 session hijacking](https://www.infostealers.com/article/compromising-google-accounts-malwares-exploiting-undocumented-oauth2-functionality-for-session-hijacking/)
- [SpyCloud — Infostealers bypass Chrome app-bound encryption](https://spycloud.com/blog/infostealers-bypass-new-chrome-security-feature/)
- [Brandefense — MFA doesn't protect you: cookies give you away](https://brandefense.io/blog/mfa-doesnt-protect-you-cookies-give-you-away-the-rise-of-session-hijacking/)
- [Obsidian Security — Session Hijacking](https://www.obsidiansecurity.com/blog/session-hijacking-how-it-works-how-to-stop-it)

### 2SV / phishing-resistant factors
- [Yubico — What is Phishing-Resistant MFA](https://www.yubico.com/resources/glossary/phishing-resistant-mfa/)
- [Google Titan Security Key](https://cloud.google.com/security/products/titan-security-key)
- [Google Security Blog — Strengthening 2SV with Security Key](https://security.googleblog.com/2014/10/strengthening-2-step-verification-with.html)
- [ZITADEL — How attackers bypass 2FA](https://zitadel.com/blog/2fa-bypass-attacks)

### Industry norms (money-moving SaaS)
- [Coinbase — 2-step verification (mandatory)](https://help.coinbase.com/coinbase/getting-started/getting-started-with-coinbase/2-step-verification)
- [Binance.US — 2FA FAQs (mandatory)](https://support.binance.us/en/articles/9842823-two-factor-authentication-2fa-faqs)
- [Stripe — Enable two-step authentication](https://support.stripe.com/questions/enable-two-step-authentication)
- [Stripe — SAML SSO](https://docs.stripe.com/get-started/account/sso)
- [Cloudflare — Enforce MFA (IdP or hosted)](https://developers.cloudflare.com/cloudflare-one/access-controls/policies/mfa-requirements/)

### Step-up authentication
- [Auth0 — What is step-up authentication, when to use it](https://auth0.com/blog/what-is-step-up-authentication-when-to-use-it/)
- [Ping Identity — Step-up authentication](https://www.pingidentity.com/en/resources/blog/post/step-up-authentication.html)

### Standards
- [NIST SP 800-63B (Digital Identity Guidelines — Authentication)](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [NIST SP 800-63-4 draft (2024 revision)](https://pages.nist.gov/800-63-4/sp800-63b.html)

### Google RISC / Cross-Account Protection
- [Google — Protect user accounts with Cross-Account Protection](https://developers.google.com/identity/protocols/risc)
- [Google Developers Blog — Working together to improve user security](https://developers.googleblog.com/en/working-together-to-improve-user-security/)
