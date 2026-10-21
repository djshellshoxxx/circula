# Prior Art: Mutual-Credit Networks (research notes for Circula)

Accessed 2026-10-21. Method caveat: WebFetch was unavailable in this session (DNS failure on all hosts), so every claim below comes from WebSearch result summaries, not from reading the primary pages in full. Source URLs are those the search results returned. Claims marked **[UNVERIFIED]** could not be confirmed; claims marked **[INFERENCE]** are my design reasoning, not sourced fact. Section 8 lists what to verify first-hand before specs are frozen.

---

## 1. Per-system notes

### 1.1 LETS (Local Exchange Trading Systems)

**Model.** Members start at zero, trade in a local unit, and each trade posts equal and opposite amounts. IOUs are logged in a central ledger, with a directory and balances visible to all members. Borrowing and lending limits are set by community rules, so any ceiling comes from each group's own charter. Default loss is absorbed by all members (so it is mutual credit). Systems often cap negative balances and sometimes positive ones too. [1][2]

**Credit limits / negative balances.** Limits are local and not standardised. A practitioner source states that no LETS "seems to have found a satisfactory way of ensuring that members do not stay permanently in deficit", which leads members in credit to become disenchanted and leave. [3]

**Defaults / departure.** Credit Commons accounting notes a departing member's account must reach zero before closing, with surplus or deficit shared among the other account holders by a pre-arranged system. [4] No standard LETS rule was retrieved. **[UNVERIFIED: typical LETS charter wording on departure, e.g. repayment plans or write-off votes.]**

**Governance / admin.** Volunteer-run. The Scottish review found low trading volume and insufficient volunteer administration plagued many schemes, and schemes depended on a few key volunteers whose withdrawal sometimes dissolved the scheme. [5]

**Decline evidence.** Australian study: schemes may decline for never reaching critical mass; those surviving 5-6 years tend to stabilise. [5] Scottish study: some people were "money rich but time poor" or had kinship networks substituting for LETS. [5] Claimed UK counts (about 450 schemes in 1998, 303 by 2001) and a "78% dissatisfaction" figure appear only on a secondary site. **[UNVERIFIED]** [5]

**Licensing.** LETS is a concept, not software.

**Adopt / avoid.** Adopt: zero-start, symmetric postings, visible balances (with privacy options). Avoid: dependence on one volunteer treasurer; unbounded permanent deficits; limits that are social convention only and not machine-enforced.

### 1.2 Cyclos (STRO, Netherlands)

**Model.** Configurable banking-style platform used for LETS, barter, time banks, microcredit, remittances and backed currencies. [6] Business rules are organised around "account types" that define payment types and applicable fees; credit limits are configured at the account type level. [7]

**Licensing.** Older sources: GPL, developed by the NGO STROhalm. [6] One wiki mirror says GPL covers only up to v3 and v4 is proprietary. **[UNVERIFIED from a primary source]** [6] A third-party blog says Cyclos 4 PRO is licensed by STRO, with a free "Communities" edition for small schemes and a hosted service; a 2014 hosting post cites free licence up to 300 users. Old and secondary. [8][9]

**Not retrieved:** Cyclos default handling, dispute features, fee specifics, and the actual per-user/group limit mechanics. **[UNVERIFIED]**

**Adopt / avoid.** Adopt: a typed account/product model (limit, fees and payment types attached to a configurable account type, not hard-coded). Avoid: closed-core licensing that locks small communities out of forking. [INFERENCE]

### 1.3 Community Forge, CES, Credit Commons, Mutual Credit Services

**Community Forge.** Drupal-based, built on the `mutual_credit` module by Matthew Slater, co-founder with Tim Jenkin's circle (the bio ties it to Geneva LETS); hosts roughly 200-300 mostly francophone groups free. Code described in 2010 as maintained by one individual. [10][11] **[UNVERIFIED: whether the hosting still operates, and the module's current licence.]**

**Community Exchange System (CES).** Founded by Tim Jenkin in 2002; sources report 340+ exchanges in 34 countries (2011) and a later count of 921 exchanges in 86 countries. Each group sets its own rules; the central site mainly does account keeping and advertising. Mutual credit; can also be configured as a time bank. [12][13] **[UNVERIFIED: CES credit-limit mechanics and licence.]**

**Credit Commons Protocol.** Open-source, tech-agnostic specification: a shared ledger for mutual-credit accounting in any agreed unit (fiat, hours, kWh), with ledgers recursively nestable so trading blocs of any scale can exchange without money. White paper by Slater and Jenkin 2016; REST protocol and reference implementation 2020; Mutual Credit Services (Dil Green) has deployed it twice. Governance analogy: like chess federation stewarding rules while implementations vary. Altered versions may not interoperate. Hosted on GitLab; the credit-commons website URL was out of date as of a 2025 interview. These are largely promoter claims. [14][15]

**Credit Commons accounting concepts.** Credit limit (how far negative a participant can go, based on what they can realistically offer); a "negative clearing limit" is the minimum balance a member must return to at the end of each settlement period; a centrally-held deposit or other collateral can secure a balance at that limit without unsecured risk to others; departing member's account must reach zero. [4][16]

**Adopt / avoid.** Adopt: protocol-first design with a documented ledger API, a nesting/federation story, collateral-backed limits as an option. Avoid: reliance on a single maintainer and a lapsing domain/docs (explicit bus-factor lesson).

### 1.4 Sardex and business mutual-credit networks

**Model.** B2B network in Sardinia; accounts open at zero, members may go negative up to a limit, like an overdraft but with no interest. [17][18] Limit is set at joining and roughly tied to what a firm can offer; reported figures: about 1% of annual turnover for negative limit and about 10% for a positive-balance cap (Sardex, unlike many mutual credit systems, caps positive balances). Sardex does not publish individual/aggregate limits. [17][19] **[UNVERIFIED: the 1%/10% figures come via a search summary of Monneta's page; confirm against Sartori & Dini 2016.]**

**Negotiation and transparency.** Each member negotiates its upper debt limit; credit status and limits are visible to other members online. [18]

**Repayment.** Negative balances carry no interest and must be recovered through sales within 12 months or repaid in euros; membership contract commits firms to avoid persistent large positive or negative balances. [19][20]

**Fees.** Graded annual membership fee (cited as EUR 300-3,000 in one presentation; EUR 200-3,000 on Wikipedia, so lower bound conflicts) rather than transaction fees. [18][21]

**Not retrieved:** broker role, insolvency handling, dispute process, and legal structure. **[UNVERIFIED]**

**Licensing.** Proprietary operator; not open source. **[UNVERIFIED but no evidence of open code.]**

**Adopt / avoid.** Adopt: symmetric caps (positive cap discourages hoarding), a time horizon on debt, public limit visibility, contractual commitment to avoid persistent imbalance. Avoid: euro-repayment as sole backstop in a no-fiat network; opaque limit-setting.

### 1.5 WIR (Switzerland)

**Model.** Founded 1934 by entrepreneurs after the interwar financial crisis; operates as a cooperative issuing credit in WIR francs to members. Credit lines historically secured by pledged assets, later also cash deposits. [22][23] Research (Studer 1998, Stodder 2009) finds turnover is countercyclical, helping small firms in downturns. [22]

**Fees / membership.** A fee is taken on WIR receipts (per a blog account). [22] Voting membership in the cooperative requires at least ten shares that are blocked while the membership lasts. [22] **[UNVERIFIED: current fee schedule and credit limit rules.]** WIR Bank's current published documents describe an unsecured "Sofortkredit" with an immediate limit of 10,000 CHW, and investment loans from 10,000 CHW up to 5 years; these are bank products and not representative of the whole system. [24]

**Licensing.** Proprietary regulated bank.

**Adopt / avoid.** Adopt: optional collateral-secured limits for large credit; countercyclical framing. Avoid: bank-regulatory structure (out of scope for an open platform).

### 1.6 Time banks (TimeOverflow, hOurworld, Timebanks.org / TimeBanks USA)

**TimeOverflow.** Open-source (AGPL-3.0 listed in directory metadata; one mirror snapshot shows another licence, so check the repo), Ruby, multi-bank per installation, with superadmin/admin/member roles; the ADBdT association offers it free to time banks. Last push listed May 2024, about 143 stars (possibly low activity). [25] **[UNVERIFIED: licence history, geography (Spain), current maintenance.]**

**hOurworld.** Nonprofit offering basic timebank software and hosting free, per c.2012 sources; current fees not confirmed. [26]

**TimeBanks USA.** Proprietary software with an ongoing fee based on membership size; separate organisation. [26]

**Timebanking UK.** Membership fee GBP 1,500 per year after first year for larger orgs (published figure on its site). [27]

**Model and failure evidence.** One-hour-one-credit equality is the norm [INFERENCE from general time-bank practice; not directly retrieved]. A Swedish study found problems in how the credit system worked, lack of long-term participants, time shortages and segregation. [28] Reports note coordinator burnout is under-studied; volunteer retention is a constant challenge (general nonprofit finding). [28] A 2025 Edinburgh report has sections on balancing time-rich and time-poor members and responsible use of exchanges; contents not read. [28][29]

**Adopt / avoid.** Adopt: multi-tenant admin roles, free hosting option for tiny groups. Avoid: per-member pricing that penalises growth; neglecting coordinator tooling.

### 1.7 Grassroots Economics / Sarafu (Kenya)

**Model.** Community Inclusion Currencies (CICs): vouchers issued by communities and backed by members' pledged goods/services; later "commitment pooling" (dated to 2022, based on indigenous rotational labour practices) where people use vouchers and pools to access zero-interest, in-kind repayable credit. Expired group vouchers are automatically deposited into a community fund usable for loans or shared needs. [30][31]

**Evidence.** A randomised trial reports $147,492 in CIC transfers across over 40,000 registered users in Kenya, 2018-2020. [32]

**Technology / licensing.** Self-described open source community-run dapp connected to Celo (earlier "Kitabu" chain); stack described as React/Flask/Postgres; repos on a self-hosted Gitea (git.grassecon.net). [31][33] **[UNVERIFIED: licence.]**

**Adopt / avoid.** Adopt: demurrage-like expiry flowing to a community fund as an anti-hoarding mechanism; pooled commitments as an alternative to personal overdraft. Avoid: blockchain dependency for a minimum viable product, which adds key-management burden for non-technical members [INFERENCE].

### 1.8 Open Money

**Model.** Project to develop software and infrastructure for peer-based multiple local currencies; associated with Michael Linton (inventor of LETS); a demo server (openmoney.cc) supports timebank or other currency forms with a wallet. [34][35] Open Co-op UK's advisory group includes Les Moore of Open Money UK; Open Co-op and LowImpact are building a UK mutual credit network with interest-free credit among co-ops/SMEs and a bespoke directory plus accounting system. [35] **[UNVERIFIED: licence, current activity, credit rules.]** The search found nothing for "OpenMoney.net" or "Gareth Scott"; those names from the brief are not confirmed.

### 1.9 Ripple-style trust-line networks

**Model.** Ryan Fugger's original Ripple (2004; RipplePay 2005): each user sets how much credit they extend to each contact (trust line); payments route through chains of trusted intermediaries who act as guarantors; balances shift along credit lines, so all money is explicitly debt. [36][37] Fugger maintains classic.ripplepay.com and villages.cc; one assessment says classic Ripple never took off. [36] Trustlines Network is a later Ethereum-based implementation. [36] Ripple Labs/RippleNet is a separate payments company. [36]

**Limits.** Per-edge, bilateral, set by users; risk is localised to those who extended the line (no network-wide shared default). [36] **[UNVERIFIED: rippling/fee/dispute details.]**

**Adopt / avoid.** Adopt: optional bilateral trust-line limits as a layer on top of network limits; pathfinding only if scope allows. Avoid: multi-hop routing in v1 (liquidity and UX complexity; adoption evidence is poor).

---

## 2. Synthesis

### 2.1 Common credit-limit policy patterns and defaults

| Pattern | Where seen | Notes |
|---|---|---|
| Uniform flat limit | General mutual credit overview | One of three common approaches [38] |
| History-based, adjusted | Same | Limits reduced for persistent deep negatives [38] |
| Turnover-/capacity-based | Sardex (about 1% turnover negative, about 10% positive cap) | Needs member declaration [17] |
| Peer/trust-line | Ripple | Risk bilateral [36] |
| Collateral-backed | WIR; Credit Commons deposit option | [4][22] |
| Settlement-period floor | Credit Commons "negative clearing limit" | [4] |

A single "typical default" number was not found in retrieved sources. **[UNVERIFIED]** Reasonable starting point for a spec is a per-network configurable default, with limits expressed in the network's unit and a recommended pattern of low initial limit plus raise-by-history. [INFERENCE]

### 2.2 Default and member-departure handling

Retrieved patterns: (a) loss socialised across members (the textbook mutual-credit definition, though "may not be even or managed") [2]; (b) account must reach zero before closing, with a pre-agreed sharing rule [4]; (c) time-boxed debt, recover through sales within 12 months or repay in fiat [19][20]; (d) collateral/deposit securing the limit [4]; (e) limit reduction for persistent deep negatives [38]. Circula should make the departure rule an explicit, versioned network policy and not an implicit norm.

### 2.3 Dispute patterns

The retrieved sources contained little on dispute mechanics: Cyclos, Sardex and CES dispute processes were not found. **[UNVERIFIED]** What is known: CES and Credit Commons delegate rules to each group [12][14]; Credit Commons uses a transaction workflow in its protocol **[UNVERIFIED: pending/confirm states]**. Common design in practice, not sourced here: payer-confirms or payee-initiates with mutual approval, then admin/committee mediation. [INFERENCE]

### 2.4 Documented failure modes and mitigations

| Failure mode | Evidence | Mitigation for Circula |
|---|---|---|
| Critical mass / thin liquidity | Australian LETS study [5] | Directory with offers/needs matching; onboarding quotas for offers |
| Permanent deficits; creditors disenchanted | [3] | Time-boxed debt; positive cap; limit tightening; visible balances |
| Volunteer-admin burnout, key-person risk | [5][28] | Role delegation, audit log, automation of routine tasks, succession checklist |
| Trust deficit among strangers | [5] | Vouching, graduated limits, public profiles |
| Offers/needs mismatch; "time-rich/money-rich" disengagement | [5][29] | Lightweight needs board; non-trade contribution paths |
| Hoarding | Sardex positive cap [19]; Sarafu expiry [30] | Optional positive cap; optional demurrage-to-community-fund |
| Inactivity / dormant members | Swedish time bank (few long-term participants) [28] | Dormancy detection, limit decay, reactivation prompts |
| Fraud / collusive balance inflation | No source retrieved **[UNVERIFIED]** | Velocity and counterparty-concentration alerts; new-member limits; dual approval for large trades [INFERENCE] |
| Single-maintainer software | Community Forge, Credit Commons docs lapse [10][14] | Open governance, documented protocol, multiple maintainers |
| Lack of institutional recognition | Essay (commentary) [5] | Out of scope; provide tax-reporting exports |

### 2.5 Feature gaps Circula could fill

Based on what was found (and not found): none of the retrieved sources show a single open, maintained tool combining (1) per-network policy-as-configuration for limits, departure and disputes; (2) automated limit review with member-visible reasons; (3) built-in dispute workflow with an evidence trail; (4) exit/close-out workflow with a ledger-neutral write-off or sharing step; (5) health dashboards (liquidity, concentration, dormancy) for admins; (6) federation between networks (Credit Commons has the protocol; adoption claims are promoter-sourced [14]). Verify against current Cyclos and CES feature sets before claiming novelty. **[UNVERIFIED]**

---

## 3. Recommendations for Circula's specs

1. **Double-entry, integer-unit ledger, append-only.** Every trade posts equal and opposite entries; corrections are reversing entries, never edits. [1][INFERENCE]
2. **Policy as versioned configuration per network.** Limits, fees, dispute and departure rules are data with effective dates and change history (matches each-network-sets-rules practice [12][14]).
3. **Pluggable limit strategies:** flat, history-based, turnover/capacity-based, collateral-backed, per-edge trust line. Ship flat and history-based first. [38][17][4][36]
4. **Asymmetric limits:** separate negative limit and optional positive cap (Sardex model) to reduce hoarding. [19]
5. **Debt time horizon:** optional maximum age for negative balance with escalation steps (reminder, limit freeze, admin review). [19][20]
6. **Public-by-default limits and balances, with a per-network privacy setting.** Transparency supports trust (Sardex, LETS). [1][18]
7. **Departure/close-out workflow:** account must reach zero or trigger the network's chosen rule (members share loss, reserve fund absorbs, collateral called); preview the effect on every member. [4]
8. **Optional reserve/collateral fund** funded by fees or deposits, so default does not always fall on members. [4][22]
9. **Dispute workflow:** pending/confirmed trade states, structured dispute with evidence, time limits, admin or committee resolution, reversal by compensating entry. Design from first principles because sources lacked mechanics. [INFERENCE]
10. **Admin-burnout controls:** role separation (treasurer, mediator, onboarder), term limits, audit log, automated reminders, and a documented handover export. [5][28]
11. **Health dashboard:** aggregate liquidity, balance distribution, dormancy, counterparty concentration, per-member trend; alert thresholds are network-configurable.
12. **Anti-hoarding and dormancy tools (optional):** demurrage or expiry feeding a community fund (Sarafu) and limit decay for dormant accounts. [30]
13. **Protocol-first and federation-ready:** documented REST API and a ledger spec compatible in spirit with Credit Commons (nestable ledgers, per-account limits) so networks can interconnect. [14]
14. **Open licence (AGPL-3.0 recommended; decision for the founders) and open governance** with at least two maintainers; avoid features gated behind commercial licences (Cyclos concerns). [6][8][10]
15. **Onboarding with graduated trust:** low starting limit, vouching by existing members, offers required before borrowing beyond a threshold, and a member agreement stating commitment to avoid persistent imbalance (Sardex contract). [19][5]
16. **Avoid in v1:** blockchain dependency, multi-hop routing, fiat conversion/redemption (regulatory exposure). [INFERENCE; legal review needed]

---

## 4. Unverified items to check first-hand

- Cyclos current licence and its credit-limit, dispute and fee features (docs.cyclos.org, STRO).
- Sardex limit formula (Sartori & Dini 2016, Stato e Mercato 107); the JRFM 2020 paper (13(12), 295) I was pointed to did not appear in searches; confirm it exists and what it says.
- WIR credit limit rules and fee schedule (wir.ch).
- Sarafu/Grassroots Economics code licence (git.grassecon.net repos).
- TimeOverflow licence history and maintenance state (github.com/coopdevs/timeoverflow).
- hOurworld current pricing (hourworld.org).
- Open Money licence/activity; "OpenMoney.net" and "Gareth Scott" from the brief were not found.
- Credit Commons: transaction states, dispute features, current repository location.
- Any quantified fraud cases in mutual credit.
- Numerical LETS decline statistics (UK 450 to 303; 78%) from primary sources.

---

## Sources (accessed 2026-10-21)

1. https://en.wikipedia.org/wiki/Local_exchange_trading_system
2. https://en.wikipedia.org/wiki/Mutual_credit
3. https://wiki.p2pfoundation.net/LETS
4. https://wiki.p2pfoundation.net/Credit_Commons_Accounting
5. https://www.researchgate.net/publication/258028332_A_REVIEW_OF_LOCAL_EXCHANGE_AND_TRADING_SCHEMES_LETS_AND_TIME_BANKS_IN_SCOTLAND ; https://www.researchgate.net/publication/253269405_Local_Exchange_And_Trading_Systems_LETS_In_Australia_A_New_Tool_For_Community_Development ; https://whatcouldhavebeenarchive.substack.com/p/what-could-have-been-local-exchange ; https://grokipedia.com/page/Local_exchange_trading_system
6. https://wiki.p2pfoundation.net/Cyclos ; https://matslats.net/node/139
7. https://cyclos.sourceforge.net/Business_rules.html
8. https://massivegrid.com/blog/cyclos-community-currency-payment-platform-paas/
9. https://portal.jvmhost.com/news/view/59/cyclos-4-hosting-at-jvm-host---free-up-to-300-users-/
10. https://groups.drupal.org/taxonomy/term/67358
11. https://wiki.p2pfoundation.net/Matthew_Slater ; https://www.ijccr.net/article/download/9136/8886/25110
12. https://beyondmoney.net/2011/12/20/ces-a-prototype-global-exchange-system-for-the-21st-century ; https://beyondmoney.net/tag/tim-jenkin
13. https://www.ashoka.org/en/fellow/timothy-jenkin ; https://wiki.p2pfoundation.net/Community_Exchange_Systems
14. https://wiki.p2pfoundation.net/Credit_Commons_Protocol ; https://wiki.p2pfoundation.net/Matthew_Slater_and_Dil_Green_on_the_Credit_Commons_Protocol
15. https://www.lowimpact.org/posts/mutual-credit-services-keeping-communities-alive-after-covid-explaining-the-credit-commons-protocol/
16. https://hackmd.io/@MCanalytics/S1KXE0HFw
17. https://monneta.org/en/sardex/
18. https://wiki.p2pfoundation.net/Sardex
19. https://www.researchgate.net/publication/316872886_Self-Funded_Social_Impact_Investment_An_Interdisciplinary_Analysis_of_the_Sardex_Mutual_Credit_System
20. https://ideas.repec.org/a/taf/jsocen/v8y2017i2p149-164.html
21. https://ancitoscana.it/wp-content/uploads/2024/02/SavoirFaire_presentazione_Sardex_Dini.pdf
22. https://en.wikipedia.org/wiki/WIR_Bank ; https://blog.p2pfoundation.net/the-wir-economic-circle-cooperative-in-switzerland-a-status-report/ ; https://beyondmoney.net/research-and-reviews/wir-current-operational-realities/
23. https://www.cgaa.org/article/wir-bank
24. https://lp.wir.ch/fileadmin/user_upload/Einlageblatt-kredite-bank-wir-de.pdf
25. https://awesome.ecosyste.ms/projects/github.com%2Fcoopdevs%2Ftimeoverflow
26. https://wiki.p2pfoundation.net/Timebanking_Software_Platforms ; https://wiki.p2pfoundation.net/HOurworld ; https://wiki.p2pfoundation.net/TimeBanks_USA
27. https://timebanking.org/join-timebanking-uk-turnover-over-500k/
28. https://www.ijccr.net/article/view/9112 ; https://blogs.otago.ac.nz/time-banking-nz/
29. https://timebanking.org/wp-content/uploads/2025/01/TimebankingTechnologyInterviewsReport-FullDraft-January2025.pdf
30. https://grassrootseconomics.org/community-voucher ; https://forum.celo.org/t/introducing-the-development-of-sarafu-wallet-a-public-good-wallet-for-celo/12402
31. https://grassrootseconomics.org/pages/sarafu-network
32. https://www.frontiersin.org/journals/blockchain/articles/10.3389/fbloc.2021.739751/epub
33. https://git.grassecon.net/grassrootseconomics/cic-docs/blame/branch/lash/initial-review/README.md
34. https://wiki.p2pfoundation.net/Open_Money
35. https://open.coop/mutual-credit/ ; https://www.loomio.com/d/coYfnUPA/webinar-proposal-mutual-credit-rea-accounting
36. https://bitcoinmagazine.com/articles/introducing-ripple/ ; https://blog.plan99.net/how-did-classical-ripple-work-13fefe1870cf ; https://wiki.p2pfoundation.net/Trustlines_Network
37. https://groups.google.com/g/rippleusers/c/eUocylJbU-U
38. https://gitcoin.co/mechanisms/mutual-credit
