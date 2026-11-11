# Additional attractiveness/adoption ideas (pre-specification inventory)

Date: 2026-11-10 (America/Vancouver). Status: **future proposal only; no implementation claim**.
These eight proposals extend the previously requested 15 future feature specifications. They are small scoping notes rather than approved work, and each requires an owner-approved full spec and ADR if selected. [See upstream porting research](UPSTREAM_PORTING_RESEARCH.md).

## ADOPT-301: Participating merchant directory

**Purpose.** Show nearby or network-available merchants and what they accept in credits.

**Minimum proposed behavior**

1. Merchant opts in to searchable business profile, with business type, verified membership, hours, broad area, categories and contact preference.
2. Search and category filters reuse member/listing permissions and paging; merchant cannot imply that Circula guarantees acceptance of credits.
3. A profile links to member's listings and a reviewed proposed trade, not a new checkout API.

**Proposed data:** merchant_profiles(network_id,member_id,display_name,category,hours,area_id,contact_policy,visibility,verification_state)

**Reuse assessment:** Medusa catalog UX (MIT core, patterns only), existing Circula directory (native).

**Minimum acceptance:** Merchant opt-out hides profile promptly; cross-network search returns nothing; closed merchant listing cannot be traded.

**Status:** future only; no code imported.

## ADOPT-302: Spending opportunities dashboard

**Purpose.** Make earned credits useful by surfacing real offers and contacts.

**Minimum proposed behavior**

1. Show active listings filterable by selected category, network area and available credit without revealing the member's balance to sellers.
2. Suggest affordable offers using the viewer's own available balance computed from authoritative ledger; suggestions never guarantee stock or acceptance.
3. Provide simple saved searches and manual refresh; do not track purchases, ad profiles or other-network histories.

**Proposed data:** Saved dashboard filters only; reuse existing listings and signed account data.

**Reuse assessment:** Existing listings; Bleve (Apache2 optional); Chart.js (MIT for optional visualization).

**Minimum acceptance:** A lower available balance or reserved outgoing proposal changes recommendations without leaking private balance; no stale completed listing.

**Status:** future only; no code imported.

## ADOPT-303: Business storefront pages

**Purpose.** Offer participating merchants a professional, small catalog using their existing listings.

**Minimum proposed behavior**

1. Public-to-network storefront composed of merchant-selected listings, photos, status, broad pickup area and network-specific shop policies.
2. A storefront link creates an ordinary proposal and acceptance; no cart, custody, third-party payments or arbitrary price conversion.
3. Business can archive products or pause storefront; transactions already posted remain unchanged.

**Proposed data:** storefront_preferences(network_id,member_id,description,theme_key,visibility), plus existing listings.

**Reuse assessment:** Medusa MIT open-source commerce modules as UX reference, not direct Node backend; imaging (MIT) for thumbnails.

**Minimum acceptance:** Only active member-owned listings appear; malicious descriptions are escaped; invalid stock/offline storefront never creates transfer.

**Status:** future only; no code imported.

## ADOPT-304: Volunteer trade coordinator tools

**Purpose.** Enable a trusted coordinator to help participants find exchanges with explicit member consent.

**Minimum proposed behavior**

1. Coordinator role scoped to one network, configurable view of consented listings and requests with no access to other private messages/ledger detail.
2. Coordinator suggests participants and sends optional introductions; cannot accept trades or debit accounts on their behalf.
3. All lookup/intro actions are auditable, revocable and subject to rate limits.

**Proposed data:** coordinator_roles plus audited proposals/introductions, opt-in consent, bounded expiry.

**Reuse assessment:** Gonum graph algorithms BSD-style for matching; Circula roles/service native.

**Minimum acceptance:** Coordinator cannot bypass member acceptance, see another network, or view restricted account balances.

**Status:** future only; no code imported.

## ADOPT-305: Privacy-first customizable profiles

**Purpose.** Let members control the exact contact and trading details visible within a network.

**Minimum proposed behavior**

1. Independent toggles for name/pseudonym, merchant/contact information, approximate area, availability and direct-message discoverability.
2. Per-profile preview as another member and graceful fallback when an old listing references an anonymized account.
3. No public indexing, no external analytics and no absolute anonymity claims.

**Proposed data:** member_profile_visibility(network_id,member_id,field,scope) with secure default members-only or hidden.

**Reuse assessment:** Extend Circula member/role ACL (native). Learn accessible visibility control patterns from GOV.UK Frontend MIT.

**Minimum acceptance:** Other-network, anonymous and blocked viewers cannot see hidden fields; profile/backup data minimization verified.

**Status:** future only; no code imported.

## ADOPT-306: Per-network branding and welcome page

**Purpose.** Let a community identify and explain its own network without forking the codebase.

**Minimum proposed behavior**

1. Admin configures limited brand colors, logo, introduction, rules link, category names and home-page announcement.
2. Validate WCAG contrast and image metadata, permit safe CSS tokens only, never arbitrary remote scripts/styles.
3. Keep critical labels and balance signs consistent; do not allow network theme to disguise financial status or login-origin.

**Proposed data:** network_branding(network_id,logo_ref,brand_tokens,welcome_markdown,updated_at) with authorized revision history.

**Reuse assessment:** GOV.UK Frontend MIT components and design-system approaches; imaging MIT for logo resizing.

**Minimum acceptance:** Malicious HTML/CSS not executed; safe color and 400% zoom; network-specific branding cannot bleed into another tenant.

**Status:** future only; no code imported.

## ADOPT-307: Referral and invitation tracking

**Purpose.** Help existing members grow a network without pressuring or publicly ranking referrals.

**Minimum proposed behavior**

1. One-time invite codes already exist; extend with consented referral attribution, invite limit controls and clearly defined invite lifecycle.
2. Allow member send/share link without exposing recipient personal data or making membership automatic.
3. No monetary reward or credit issuance for recruitment by default; optional incentives require separate grant/ledger spec.

**Proposed data:** invitation_referral(invitation_id,referrer_member_id,created_at,accepted_at) with retention and private visibility.

**Reuse assessment:** Extend Circula current /api/v1/admin/invitations and membership code; no external identity system needed.

**Minimum acceptance:** Replay and brute-force invite defenses, idempotent registration, revocation and no unsolicited profile disclosure.

**Status:** future only; no code imported.

## ADOPT-308: Optional gift credits and vouchers

**Purpose.** Support bounded community-issued gifts with transparent accounting.

**Minimum proposed behavior**

1. Admin may create a single-use voucher representing a future eligible grant against the existing network system account; no promise of redemption in cash.
2. Voucher redemption requires authenticated eligible member, explicit network policy/caps, reason and two equal/opposite ledger entries.
3. Separate campaign budgets, limits, expiry, revocation, reporting and reviewer approval; never allow vouchers to bypass credit or audit rules.
4. Do not implement transferable bearer tokens or cross-network cash equivalence as part of this scope.

**Proposed data:** voucher_campaign(network_id,id,max_issued,expiry,policy_version,approval_state), voucher_token_hash(network_id,campaign_id,token_hash,redemption_tx_id) and atomic grant record.

**Reuse assessment:** Use Circula's already implemented system-account grant/adjustment transaction path; copying generic coupon code cannot enforce double-entry accounting.

**Minimum acceptance:** Parallel double-redemption produces one capped grant; invalid/expired/cross-network vouchers cannot change ledger; issuance and redeemed totals reconcile.

**Status:** future only; no code imported.

## Adoption sequencing

Start with merchant directory and spending opportunities once the underlying listing search and privacy rules are clear. Follow with business storefronts, profile visibility, network branding and referrals. Coordinator tools require granular role/consent review. Gift credits/vouchers require a separate ledger and issuance-policy security review, and are not a casual marketplace add-on.

No feature changes the ledger's balance-zero invariant, consent, network isolation, governance approvals or payment/cash-redemption scope. All changes require the same CI, migration, auth, accessibility and review gates as the primary 15 specifications.
