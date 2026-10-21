# Circula: Regulatory, Privacy and Accessibility Questions for Counsel

Status: research notes, accessed 2026-10-21. **This is not legal advice.** It is an issues list for qualified counsel in each jurisdiction, with pointers to primary sources.

## How to read this document

- **[V]** = supported by a source returned by a search in this research session (usually a secondary summary of a primary source, because direct fetches of irs.gov, fincen.gov and eba.europa.eu failed with DNS errors in the research sandbox).
- **[U]** = unverified. It comes from general background knowledge and must be confirmed against the primary source before anyone relies on it.
- Primary-source URLs in the Sources section are the places counsel should check. Several were not opened by the researcher.
- Statute and guidance numbers change. Re-check currency of every citation before launch.

## 1. Product assumptions used for the analysis

Circula, as described: members of a community network record exchanges of a unit of account (LETS-style mutual credit). Balances can go negative within a credit limit. The unit is not a cryptocurrency and is not redeemable for national currency by design. Network administrators (communities) run networks. A platform operator (the Circula project or a host) may later offer hosted subscriptions paid by members or admins.

The facts that drive most regulatory outcomes are:

1. Whether units can ever be converted to or from national currency.
2. Whether the platform operator ever holds or moves national-currency funds for members.
3. Whether the unit is accepted by third parties outside the closed membership.
4. Whether balances accrue interest, are sold, or are marketed as an investment.
5. Who decides the purposes and means of processing personal data (operator or network admin).

## 2. Survey by topic

### 2.1 Barter, LETS and mutual-credit tax treatment

**US federal.**
- The IRS treats barter as an exchange of goods or services, and says fair market value of what is received is gross income in the year of receipt [V]. Business barter income is generally reported on Schedule C [V].
- A "barter exchange" is an organization whose members contract with each other or with the organization to trade property or services. The IRS says it excludes informal arrangements such as a babysitting co-op [V]. Barter exchanges file Form 1099-B for transactions, unless an exception applies [V]. The statutory source is Internal Revenue Code section 6045(c)(3) and Treasury Regulation 1.6045-1(e) [U: cited from memory; counsel to confirm text, the "substantial" membership criteria, and any dollar thresholds].
- Key question: is a Circula network, or the operator of one, a "barter exchange"? A network with recorded business-to-business trades and administrators who keep the ledger is closer to a barter exchange than a neighbour swap. Who has the filing duty (network admin or operator) and does the platform need to collect TINs (W-9) and produce 1099-B data? [U]
- Tax on barter income for members applies regardless of whether Circula reports it [V as to the IRS position].

**UK.** The search found no HMRC guidance specifically on LETS; only forum commentary, including an argument that restricted-membership mutual schemes might be "mutual trading" outside tax, which is an unsettled practitioner view and not an HMRC position [V as to what the sources say; the position itself is **unverified**]. HMRC manual sections on barter and trading income, and on VAT treatment of non-monetary consideration, should be checked [U]. General principle to confirm: non-cash consideration can still be taxable income, and VAT may apply to supplies for non-monetary consideration by VAT-registered traders [U].

**EU / Canada / Australia / NZ.**
- Canada: CRA has long treated barter transactions as taxable at fair market value, and GST/HST can apply to barter supplies; barter-exchange reporting is also a topic [U for all].
- Australia: the ATO treats barter and trade credits as assets other than money, so trades are generally assessable, and GST applies to taxable supplies made for non-cash consideration. An Australian LETS operator site says the ATO regards credits as not money [V as a secondary statement only]. ATO barter-exchange reporting (an annual report by barter exchange operators) [U].
- NZ: IRD treats goods or services received in barter as income at market value; GST applies [U].
- EU: VAT on supplies for non-monetary consideration is a general VAT Directive principle (Directive 2006/112/EC, Art. 2 and 73) [U], national implementations vary.

### 2.2 Money transmission, e-money, payment services, deposit-taking, credit

**US federal (FinCEN).** A money transmitter accepts value from one person and transmits value to another person or location by any means, and MSB status depends on facts and circumstances [V, from FIN-2019-G001 summaries]. FinCEN's regulations also cover issuers and sellers of prepaid access [U]. Whether accepting and transferring a closed-loop, non-redeemable unit of account is "value that substitutes for currency" is the core question [U]. FinCEN guidance on "convertible" virtual currency centres on convertibility into real currency or acting as a substitute for it [V in outline]. A unit with no redemption and no conversion path is the best fact pattern, but counsel must confirm.

**US states.** Each state has its own money transmission law (a number of states have adopted the Money Transmission Modernization Act model [U]). Definitions of "money transmission" and "stored value" vary, and several states regulate closed-loop or "scrip" differently [U]. New York "virtual currency" licensing (BitLicense) and California's Digital Financial Assets Law apply to crypto-type activity [U]. Counsel should map the states where early networks sit.

**Credit limits as lending.** Does a negative balance within a limit amount to extension of credit? Considerations: the counterparty is the community (the ledger obligation is to the network, not a lender), no interest and no fees, and the "credit" is denominated in a non-currency unit. Truth in Lending, state lending licensing, usury and UK/EU consumer credit rules are triggered mainly by credit in currency or by a lender in business [U]. This is a key question for counsel, especially if a negative balance can ever be settled in national currency on member exit or network closure.

**UK.**
- Payment Services Regulations 2017 and Electronic Money Regulations 2011 (FCA) [U for details]. A closed-system unit of account that cannot be redeemed, is accepted only within a narrow network and is not "monetary value" held in exchange for funds may fall outside e-money; the limited-network exclusion exists in PSRs Schedule 1 Part 2 [U].
- Consumer credit: Consumer Credit Act 1974 and FCA regime for lending and "credit broking" [U].
- Banking Act / FSMA "deposit-taking" is triggered by accepting deposits of money. A mutual credit unit is not a deposit unless funds are taken in [U].

**EU.**
- PSD2 Article 3(k) excludes instruments usable only within the issuer's premises, within a limited network of providers under a direct commercial agreement with a professional issuer, or for a very limited range of goods or services. Article 37(2) requires notification where the 12-month transaction value exceeds EUR 1 million [V]. The EBA issued final guidelines on the exclusion on 24 February 2022, because member states applied it inconsistently [V]. National authorities decide, not the provider's self-assessment [V, from a consultation response].
- E-Money Directive 2009/110/EC: e-money is monetary value issued on receipt of funds and redeemable at par [U]. A unit never issued against funds and never redeemable should not be e-money, but "issued against a subscription fee" facts need review.
- PSD3 / Payment Services Regulation: proposals were published in 2023 and may have been adopted or modified by now. The search returned nothing on how they treat limited networks [**unverified; check current status**].
- Consumer Credit Directive (EU) 2023/2225 (applies from November 2026 [U]) may cover credit-like arrangements; confirm scope and exemptions for interest-free, no-fee credit [U].
- MiCA (Regulation (EU) 2023/1114): applies to crypto-assets that can be transferred and stored using DLT or similar technology. Recital 17 and Article 2 carve-outs, including for non-fungible and non-transferable assets, exist [U]. A database ledger of non-transferable (to outsiders) units on a conventional server is outside the usual understanding of "crypto-asset" if no DLT is used and units are not freely transferable, but counsel should confirm. If a future version anchors the ledger to a blockchain, re-run this analysis.

**Canada.** FINTRAC regulates money services businesses (MSBs) under the PCMLTFA, and since 2024 the Retail Payment Activities Act (RPAA) with the Bank of Canada supervises payment service providers [U]. Provincial consumer protection and Quebec's own regime also matter. Does a closed-loop non-redeemable unit fall inside "funds" or "virtual currency" definitions [U]?

**Australia.** AUSTRAC regulates remittance and digital currency exchange providers under the AML/CTF Act [U]. ASIC regulates "non-cash payment facilities" and managed investment schemes under the Corporations Act. Non-cash payment facility exemptions for limited-use facilities need review. The National Consumer Credit Protection Act 2009 may apply to credit [U]. Reforms to payment-services regulation have been under consultation in recent years [U].

**New Zealand.** AML/CFT Act 2009 and Financial Service Providers (Registration and Dispute Resolution) Act 2008 require registration of financial service providers including those operating a "payment system" or providing "money or value transfer" [U]. The Credit Contracts and Consumer Finance Act 2003 may apply to credit [U].

**Circula behaviours that could trigger these regimes** (summary):

| Behaviour | Risk it creates |
|---|---|
| Cash-out or redemption into national currency | Converts the unit into e-money / stored value / virtual currency; likely MSB or e-money issuer |
| Cash-in (buying units with national currency) | Same, plus prepaid access / issuer of stored value |
| Operator holding member funds (escrow, fees collected on behalf of admins) | Money transmission / payment services; safeguarding duties |
| Fees in national currency charged per trade | Treats the platform as payment-service intermediary; also barter-exchange fee income |
| Hosted subscription paid in national currency | Generally ordinary SaaS revenue, but facts matter if subscription converts into units |
| Interest on positive or negative balances | Deposit-taking / lending / securities |
| Third-party merchants accepting units for goods | Weakens "limited network" arguments; widens consumer-protection scope |
| Transferring units between networks | Interlinking may undermine closed-loop assumptions |
| Unlimited or large credit lines, or credit enforced in currency | Consumer credit / lending |

### 2.3 Securities and crypto classification

- US: the *Howey* investment-contract test (SEC v. W.J. Howey Co., 328 U.S. 293 (1946)) asks about an investment of money in a common enterprise with an expectation of profits from others' efforts [U]. Units that are earned by working, spent for goods, have a stable face value, and cannot be sold for profit, are far from that profile. Risk factors: pre-sales, marketing as appreciating, scarcity mechanics, staking, interest, revenue sharing.
- CFTC treats some virtual currencies as commodities; irrelevant if no tradable market exists [U].
- EU: MiCA categories (e-money token, asset-referenced token, utility token) apply only to crypto-assets using DLT [U].
- UK: FCA cryptoasset regime and financial promotions rules apply to "qualifying cryptoassets" [U]. The UK's cryptoasset regulatory framework was in legislative development; check status [**unverified**].
- Canada: securities regulators apply an investment-contract test (Pacific Coast Coin Exchange v. Ontario Securities Commission [U]).
- Australia: ASIC INFO 225 on crypto-assets [U]. NZ: Financial Markets Conduct Act 2013 [U].
- Avoid terminology such as "token", "coin", "mint", "stake", "yield", "airdrop" in product, marketing and code comments. This is a design guardrail, not a legal conclusion.

### 2.4 Consumer protection, terms of service, dispute resolution

- Who is the contracting party with the member: the operator, the network admin, or both? Hosted subscriptions bring standard consumer-protection duties: clear pricing, auto-renewal rules, cancellation rights, refund policy. In the US this includes FTC rules and state automatic-renewal laws such as California's [U]; the EU Consumer Rights Directive 2011/83/EU gives 14-day withdrawal rights for distance contracts, with specific rules for digital content and services [U]; UK Consumer Rights Act 2015 and the Digital Markets, Competition and Consumers Act 2024 subscription regime [U].
- Unfair-terms law: EU Directive 93/13/EEC, UK Consumer Rights Act 2015 Part 2 [U]. Limits on liability and mandatory arbitration clauses need local review. Class-action waivers may be unenforceable in some jurisdictions [U].
- Peer-to-peer trade disputes: members trade with each other, not with Circula. Terms must say the operator is not a party to trades and is not a marketplace guarantor. EU Digital Services Act marketplace duties (trader traceability, Art. 30) could apply if Circula is an "online platform" allowing consumers to conclude distance contracts with traders [U]. Platform-to-Business Regulation 2019/1150 applies if business users are offered services [U]. Counsel should check applicability.
- Platform-economy tax reporting: EU DAC7 (Directive 2021/514) obliges platform operators to report seller income. OECD Model Rules and UK equivalents exist. A mutual-credit platform where consideration is in units might still be "relevant activity" [U; important open question]. US analogues: Form 1099-K thresholds for third-party settlement organizations, which require payment in currency and probably do not apply [U].
- Dispute handling: internal complaint process, EU online dispute resolution platform (retired in 2025 [U]), alternative dispute resolution, and community governance (admin decisions on ledger corrections). Define who may reverse a transaction and on what basis.
- Network wind-up: what happens to negative and positive balances on dissolution? Contractual terms should avoid creating a debt payable in currency (see credit analysis above).

### 2.5 Privacy and data protection

**Controller vs processor (key architectural question).**
- In a hosted model, the network admin (a community group) decides why members' data is collected (to run the network), while the operator hosts the data. Typical analysis: network = controller, operator = processor for member and ledger data. But the operator is a controller for its own purposes (billing, security, product analytics, account management of admins), and may be a joint controller if it determines key means or shares data across networks (directory, federation, global search) [U; confirm against EDPB Guidelines 07/2020 on controller/processor concepts].
- Consequences: data processing agreement (GDPR Art. 28) between operator and each network admin; sub-processor chain; who answers data subject requests; whether small volunteer-run networks can realistically discharge controller duties. In self-hosted open-source mode, the operator may owe nothing, but the software's defaults still matter (privacy by design, Art. 25).
- Counsel decision: choose a default allocation (processor-for-network plus controller-for-account-data) and document it in terms and DPA templates.

**Lawful basis.** Likely Art. 6(1)(b) contract (membership) for core processing, 6(1)(f) legitimate interests for fraud prevention and security, 6(1)(a) consent for optional directory visibility or marketing, and 6(1)(c) for legal obligations such as tax records [U]. Check that members' "public directory" and ledger visibility to other members is justified and limited. Transaction visibility within a network reveals behavioural data, including inferred special-category data (for example, health-related services or religious community trades) [U].

**DPIA triggers.** Art. 35 and national lists: large-scale processing, systematic monitoring, vulnerable data subjects, innovative technology, profiling and scoring (for example, trust scores or credit-limit algorithms) [U]. Counsel to decide whether a template DPIA ships with the platform.

**Data subject rights vs immutable ledger and audit retention.**
- An append-only ledger is an accounting record. Design option: store personal data (names, contact details) in a separate mutable profile table, with ledger entries using pseudonymous member IDs, so erasure can remove identifying data while retaining the transactional record. Whether a pseudonymised record is still personal data (it remains so if re-identifiable by the controller) needs analysis [U].
- Erasure (Art. 17) has exceptions for legal obligations and establishment or defence of legal claims (Art. 17(3)(b), (e)) [U]. Retention periods for accounting records conflict with erasure demands; define and publish a schedule.
- EDPB Guidelines 02/2025 on processing personal data through blockchain technologies: version 2.0 was adopted 7 July 2026 after consultation [V, from search listing]. They discourage on-chain personal data, favour off-chain storage with commitments or references, and say that where rights cannot be honoured, controllers should consider alternative architectures [V, from secondary summaries; **verify against the PDF**]. Even if Circula uses a conventional database, the same reasoning on immutability and erasure is instructive.
- Right of access and portability (Arts. 15, 20): provide per-member export of ledger entries involving them, which also exposes counterparties' data. Define redaction rules.

**CCPA/CPRA (California).** Applies to businesses meeting revenue, data-volume or data-sale thresholds [U]. A small nonprofit network may fall below thresholds, but a hosted operator might not. Service-provider contract requirements, rights to know, delete, correct, and opt out of sale or sharing, and notice at collection are the key items [U]. Other US state privacy laws (Virginia, Colorado, Texas, and so on) have similar structures with differing thresholds [U]. Counsel to produce a state matrix.

**Children's data.** GDPR Art. 8: digital consent age 13 to 16 depending on member state (for information society services offered directly to a child, where consent is the lawful basis) [U]. UK Age Appropriate Design Code applies to services likely to be accessed by children [U]. US COPPA (under 13) [U]. Decision: set a minimum member age (for example, 18) in terms and onboarding, and decide how youth or family accounts (common in community timebanks) are handled, such as a guardian-managed sub-account.

**Cross-border transfers.** Operator hosted in one region and networks in others: EU to US transfers need SCCs or the EU-US Data Privacy Framework adequacy decision (July 2023, subject to legal challenge) [U]; UK IDTA or UK addendum and the UK-US data bridge [U]. Self-hosting reduces but does not remove transfer questions (for example, support access). Canada's PIPEDA and Quebec Law 25, Australian Privacy Act (APP 8 cross-border disclosure), and NZ Privacy Act 2020 (IPP 12) have their own rules [U].

**Breach notification timelines** (confirm each):
- GDPR Art. 33: notify supervisory authority within 72 hours of becoming aware, unless unlikely to result in risk; processors must notify controllers "without undue delay" (Art. 33(2)) [U]. UK GDPR is the same with the ICO [U].
- US: all 50 states have breach notice laws with varying deadlines; some set fixed days [U].
- Canada PIPEDA: report breaches posing a "real risk of significant harm" to the OPC "as soon as feasible" [U]. Australia: Notifiable Data Breaches scheme, assess within 30 days and notify when serious harm is likely [U]. NZ: notify the Privacy Commissioner as soon as practicable, no later than 72 hours after becoming aware of a notifiable breach (Privacy Act 2020 [U], check any amendments).

**Record retention.** Tax and accounting record retention requirements (for example, US IRS periods, UK six years for most business records, EU member state commercial law) interact with data-minimisation and storage limitation [U]. Counsel should set retention schedules per data category: ledger entries, identity data, support logs, backups, and deleted-account tombstones.

**Other privacy items.** ePrivacy / cookie rules for web clients; record of processing (Art. 30); data protection officer thresholds (Art. 37); EU representative (Art. 27) if the operator is outside the EU yet targets EU users [U].

### 2.6 Accessibility law

- European Accessibility Act (Directive (EU) 2019/882): applies from 28 June 2025 to specified consumer products and services, including e-commerce services, consumer banking services, and electronic communications; microenterprises providing services are exempt (fewer than 10 employees and turnover or balance sheet at most EUR 2 million) [U]. Whether a mutual-credit platform is an "e-commerce service" or "consumer banking service" is a classification question. Harmonised standard EN 301 549 and WCAG 2.1 AA as the practical benchmark [U]. A volunteer-run network or an open-source library may be out of scope but a commercial hosted subscription operator may not be.
- US ADA Title III: courts split on whether websites are "places of public accommodation"; litigation risk is real for commercial web services regardless [U]. DOJ's April 2024 rule adopting WCAG 2.1 AA applies to state and local government websites (Title II), which matters if municipalities or public libraries adopt Circula [U]. Section 508 applies to federal agencies and their procurement, so it matters only for federal customers [U].
- UK Equality Act 2010 duty to make reasonable adjustments for service providers [U].
- Canada: Accessible Canada Act (federal sector) and provincial laws such as the AODA in Ontario [U]. Australia: Disability Discrimination Act 1992 [U]. NZ: guidance and Human Rights Act 1993 [U].
- Decision: adopt WCAG 2.2 AA as the engineering target and publish an accessibility statement.

## 3. Questions for Counsel

Source column uses short names defined in the Sources section.

| ID | Topic | Why it matters | Circula behaviour that triggers it | Jurisdictions | Decision needed before launch | Source |
|---|---|---|---|---|---|---|
| LEG-Q-01 | Is a Circula network or operator a "barter exchange" for 1099-B? | Filing duty, TIN collection, penalties | Admin-run ledger of member-to-member trades, especially business members | US federal | Who files (admin or operator), thresholds, whether to build 1099-B export and W-9 collection | IRS-1, IRS-2, CFR-1 |
| LEG-Q-02 | Member income tax on units earned | Members may owe tax on FMV of units received; users will ask | Earning units for services | US, UK, CA, AU, NZ, EU | Wording of tax disclaimers; whether to display an indicative currency valuation (which conflicts with "no fiat link") | IRS-1, IRS-3 |
| LEG-Q-03 | VAT/GST on trades in units | Registered traders may owe VAT/GST on non-cash supplies; invoicing duties | Business members trading in units | UK, EU, CA, AU, NZ | Whether to add invoice/tax-code fields; terms allocating tax responsibility to members | HMRC-1, ATO-1, VATD |
| LEG-Q-04 | UK treatment of LETS by HMRC (no authoritative guidance found) | Mutual-trading argument is unsettled | Any UK network | UK | Obtain written view or adviser opinion; publish guidance for UK admins | HMRC-1 (search found only forum commentary) |
| LEG-Q-05 | DAC7 / platform reporting | Operators may need to report seller activity to tax authorities | Hosted platform facilitating member sales | EU, UK (equivalent), OECD | Whether unit-denominated trades are reportable; data fields to collect | DAC7 |
| LEG-Q-06 | FinCEN MSB: money transmitter or prepaid access? | Federal registration, AML program, SAR duties | Any redemption, cash-in, operator holding of funds | US federal | Written confirmation that closed-loop, non-redeemable units with no fiat flow fall outside | FIN-1, FIN-2 |
| LEG-Q-07 | State money transmission / stored value laws | Licensing is state by state, with criminal penalties for unlicensed activity | Fees or funds flowing through operator; any cash-in/out | US states, notably NY, CA, TX, FL | State matrix; which states to geo-fence in hosted offering | MTMA, state regulators |
| LEG-Q-08 | Do credit limits amount to lending? | Lending licences, TILA, usury, consumer-credit regimes | Negative balances within a limit; settlement in currency on exit | US, UK, EU, CA, AU, NZ | Whether negative balances may ever be enforced in currency; wording of credit-limit terms; interest prohibited | CCD2, FCA-CC |
| LEG-Q-09 | Deposit-taking | Taking deposits is a regulated activity | Positive balances being treated as owed money | UK, EU, US, AU | Confirm positive unit balances are not "deposits" and terms say so | FSMA |
| LEG-Q-10 | PSD2 limited-network exclusion / e-money | Unauthorised payment service or e-money issuance; EUR 1m notification trigger | EU networks with many members and merchants; any link to funds | EU member states | Whether exclusion or "not a payment service" analysis applies; whether to notify NCAs; EUR 1m volume monitoring | PSD2, EBA-LN |
| LEG-Q-11 | PSD3/PSR status and changes | Rules for limited networks may tighten | Same as above | EU | Confirm current legal state at launch | EU-PAY (unverified) |
| LEG-Q-12 | UK PSRs 2017 / EMRs 2011 limited-network exclusion and FCA stance | Same as EU, UK-specific rules | UK networks | UK | Same | FCA-PSR |
| LEG-Q-13 | Canada RPAA / FINTRAC | Payment service provider registration; MSB registration | Any funds flow; possibly "virtual currency" definition | Canada | Confirm closed-loop non-redeemable units are out of scope | FINTRAC, BOC |
| LEG-Q-14 | Australia: AUSTRAC, ASIC non-cash payment facility, credit | Registration, licensing, AFSL | Any funds flow; credit | Australia | Confirm limited-use exemptions; ATO barter reporting duties | AUSTRAC, ASIC |
| LEG-Q-15 | NZ: FSP registration, AML/CFT, CCCFA | Registration even for small providers | Any "payment system" or value transfer service | New Zealand | Confirm out of scope or register | FSPR |
| LEG-Q-16 | Fees in national currency | Currency fee on a ledger service may turn the operator into an intermediary or the network into a barter exchange with fee income | Per-trade or membership fees paid in fiat to admins/operator | All | MVP: fiat only for software subscription, never per trade, never deducted from balances | various |
| LEG-Q-17 | Hosted subscription structure | Revenue characterisation; consumer-protection duties; sales tax/VAT on SaaS | Admin or member pays operator in fiat | All | Who is the customer (admin vs member); ToS; VAT/sales-tax registration | CRD, DMCCA |
| LEG-Q-18 | Securities / investment contract risk | Securities law consequences if units look like investments | Marketing, scarcity, appreciation, interest, resale | US, CA, UK, AU, NZ | Marketing and copy guidelines; prohibit trading units for profit | HOWEY, SEC |
| LEG-Q-19 | MiCA / crypto-asset scope if ledger ever uses DLT | MiCA authorisation, white-paper rules | Any blockchain anchoring or tokenised representation | EU, UK | Decide DLT is out of scope for MVP; re-run analysis before any change | MiCA |
| LEG-Q-20 | Terms of service and liability allocation (operator vs admin vs member) | Enforceability; unfair-terms review | All users | All | Contract structure; mandatory-law carve-outs; governing law; arbitration/class waiver | UCTD, CRA |
| LEG-Q-21 | Marketplace / platform duties (DSA Art. 30, P2B) | Trader traceability, notice-and-action, complaint handling | Hosted directory where members offer goods/services | EU | Whether hosted offering is an "online platform"; size exemptions | DSA |
| LEG-Q-22 | Dispute resolution and reversals | Governance of ledger corrections, member disputes | Admin reversal of trades; chargeback-like expectations | All | Written dispute process, appeal, who may amend ledger, audit trail | internal |
| LEG-Q-23 | Wind-up and exit of balances | Debt characterisation; negative balances; insolvency of network entity | Network closing; member leaving with positive or negative balance | All | Terms: balances lapse or are not enforceable in currency; confirm enforceability | internal |
| LEG-Q-24 | Controller / processor allocation (operator vs network admin) | Determines DPA, rights handling, liability, regulator exposure | Hosted multi-network service; any cross-network directory | EU, UK, others | Adopt default model, DPA template, joint-controller arrangement if federated | EDPB-CP, ICO-CP |
| LEG-Q-25 | Lawful basis and transparency | Wrong basis creates invalid processing | Member profile, ledger visibility, directory | EU, UK | Basis per purpose; privacy notice templates for admins | GDPR |
| LEG-Q-26 | DPIA requirement and template | Mandatory for high-risk processing | Trust scores, credit-limit logic, large networks | EU, UK | Whether DPIA is mandatory for operator; ship template for admins | GDPR-35, EDPB-DPIA |
| LEG-Q-27 | Erasure/rectification vs append-only ledger and audit retention | Core design tension | Immutable or hash-chained entries; audit logs | EU, UK, CA, others | Approve pseudonymous-ID design; retention schedule; anonymisation procedure; tombstoning | GDPR-17, EDPB-BC |
| LEG-Q-28 | Subject access and portability with counterparty data | Exports expose other members | Member data export | EU, UK, CA | Redaction rules and export format | GDPR-15 |
| LEG-Q-29 | CCPA/CPRA and other US state privacy laws | Thresholds, service-provider contracts | Hosted operator with California members | US states | Applicability analysis; service-provider addendum | CCPA |
| LEG-Q-30 | Children and minors | Special consent and design duties; COPPA | Family or youth accounts in timebank-style networks | EU, UK, US, others | Minimum age 18 in terms vs guardian sub-accounts | GDPR-8, COPPA, AADC |
| LEG-Q-31 | Cross-border transfers | Transfer mechanisms and local-law risk | Hosting region; support access; sub-processors | EU, UK, CA, AU, NZ | Hosting regions; SCC/IDTA/DPF reliance; transfer impact assessment | GDPR-V |
| LEG-Q-32 | Breach notification and processor duties | 72-hour clocks; processor must alert controllers promptly | Security incident at operator | EU, UK, US states, CA, AU, NZ | Incident response plan; contractual notice time (for example, 24 to 48 hours to admins) | GDPR-33, OPC, OAIC |
| LEG-Q-33 | Record retention periods | Tax/accounting retention vs minimisation | Ledger, invoices, identity | All | Retention schedule per data category | tax authorities |
| LEG-Q-34 | Accessibility: EAA, ADA, Equality Act | Statutory exposure and litigation risk for commercial hosted service | Hosted web app | EU, US, UK, CA, AU | Scope (microenterprise exemption?), WCAG target, accessibility statement | EAA, ADA-WEB |
| LEG-Q-35 | AML/sanctions screening | Even if not an MSB, sanctions law applies to everyone; large unit volumes may attract scrutiny | Onboarding members across borders | US (OFAC), UK, EU | Whether any screening is required; geo-restrictions for hosted service | OFAC |

## 4. Design guardrails that reduce regulatory exposure

These are technology-neutral product and policy choices. They reduce risk but do not remove the need for counsel.

1. **No redemption, no cash-out.** Units cannot be exchanged for national currency by the platform, by default admin tooling, or via platform-facilitated third parties. Terms say so plainly.
2. **No cash-in.** Units are created only by recorded exchanges (and an initial zero balance), never sold for money.
3. **No interest and no demurrage-as-yield.** No return on balances, no payment for holding units, no rate on negative balances.
4. **No fiat fees on trades in the MVP.** The operator charges only a flat software/hosting subscription, billed outside the ledger and never deducted from, or settled with, unit balances.
5. **Operator never holds or moves member funds.** Subscription is paid to the operator for its own service through a third-party payment processor that handles card data. No escrow.
6. **Closed membership.** Units are usable only among registered members of a network; no merchant onboarding at large; no public acceptance. Admins cannot accept the unit as payment at retail outside the network.
7. **Credit limits are community-governed and non-monetary.** Negative balances are an obligation to supply goods or services to the community, with no default interest, no collection in currency by the platform, and wind-up rules that cancel residual balances (subject to counsel).
8. **Neutral terminology.** Avoid "coin", "token", "mint", "stake", "invest", "yield", "wallet" in a bank sense, "deposit", "loan". Prefer "unit", "account", "credit limit", "commitment".
9. **No investment framing or secondary market.** No marketing implying appreciation, no unit-to-unit exchange rates between networks that function as a market, no pre-sale.
10. **No distributed ledger in MVP.** Use a conventional database with an append-only, auditable ledger. Avoid public blockchain anchoring of personal data.
11. **Privacy by design.** Pseudonymous member IDs in ledger rows; personal profile data separable and deletable; minimal fields; configurable visibility of balances and transaction history; default private; purpose-limited analytics.
12. **Role clarity.** Default hosted model documented as network = controller, operator = processor (plus operator as controller for account/billing); DPA accepted at network creation; sub-processor list published.
13. **Adults only by default.** Minimum age in onboarding; guardian features only after counsel review.
14. **Export and reporting hooks, not reporting duties.** Provide per-member and per-network transaction exports so members and admins can meet their own tax obligations; do not compute or assert tax values.
15. **Transparent disclaimers without legal conclusions.** Plain-language notice that trades may be taxable and that Circula does not provide tax advice.
16. **Geofencing and feature flags.** Ability to disable hosted offerings or specific features per jurisdiction without code forks.
17. **Accessibility baseline.** WCAG 2.2 AA in design system and CI checks; accessibility statement; keyboard and screen-reader testing.
18. **Retention and deletion tooling.** Configurable retention per data category; documented anonymisation procedure for departed members.
19. **Incident response readiness.** Internal 24-hour escalation to network admins so controllers can meet 72-hour regulatory clocks.
20. **Licence and governance clarity for open source.** Software licence disclaims warranties and legal compliance; each deployer is responsible for local compliance, and docs say so.

## 5. Unverified or weakly supported items (summary)

- All statutory section numbers and thresholds marked [U] were written from memory and not checked against the text.
- HMRC's actual position on LETS was not found; only forum commentary [V as to what it says].
- PSD3/PSR current status and limited-network treatment: not found.
- Canada, Australia and NZ money-services, payment and credit regimes: not researched with sources; entirely [U].
- EDPB Guidelines 02/2025: details come from secondary summaries and search listings; the PDF (v2.0, 7 July 2026) was not opened.
- DPF adequacy status, EAA scope, and Consumer Credit Directive application date should be re-confirmed because they change.
- Direct fetches of irs.gov, fincen.gov and eba.europa.eu failed (DNS error in sandbox), so IRS and EBA summaries come from search-engine summaries.

## 6. Sources (all accessed 2026-10-21 unless noted; "search" means surfaced in a search result but not opened)

- IRS-1: IRS, Topic no. 420, Bartering income. https://www.irs.gov/taxtopics/tc420.html (search)
- IRS-2: IRS, Barter exchanges. https://www.irs.gov/businesses/small-businesses-self-employed/barter-exchanges (search)
- IRS-3: IRS Publication 525, Taxable and Nontaxable Income. https://www.irs.gov/publications/p525 (not opened; cited via secondary source)
- CFR-1: 26 CFR 1.6045-1 and IRC 6045(c)(3), via https://www.ecfr.gov and https://www.law.cornell.edu/uscode/text/26/6045 (not opened)
- FIN-1: FinCEN, FIN-2019-G001 CVC Guidance. https://www.fincen.gov/sites/default/files/2019-05/FinCEN%20CVC%20Guidance%20FINAL.pdf (search)
- FIN-2: FinCEN, Money Services Business definition. https://www.fincen.gov/money-services-business-definition (not opened, fetch failed)
- MTMA: Conference of State Bank Supervisors, Money Transmission Modernization Act. https://www.csbs.org (not opened)
- HMRC-1: HMRC manuals (search "barter" at https://www.gov.uk/hmrc-internal-manuals); secondary forum commentary: https://www.accountingweb.co.uk/any-answers/lets-barter-scheme (search)
- ATO-1: Australian LETS operator note on tax: https://brislets.com/tax-centrelink-and-gst/ (secondary, search); ATO barter pages https://www.ato.gov.au (not opened)
- VATD: Directive 2006/112/EC. https://eur-lex.europa.eu/eli/dir/2006/112/oj (not opened)
- DAC7: Directive (EU) 2021/514. https://eur-lex.europa.eu/eli/dir/2021/514/oj (not opened)
- PSD2: Directive (EU) 2015/2366. https://eur-lex.europa.eu/eli/dir/2015/2366/oj (not opened)
- EBA-LN: EBA Guidelines on the limited network exclusion under PSD2 (24 Feb 2022), summarised at https://www.loyensloeff.com/insights/news--events/news/eba-guidelines-on-the-limited-network-exclusion-under-psd2/ and https://www.regulationtomorrow.com/2022/02/eba-publishes-final-guidelines-on-the-limited-network-exclusion-under-the-psd2/ (search); EBA page https://www.eba.europa.eu (fetch failed)
- EU-PAY: Commission PSD3/PSR proposals. https://finance.ec.europa.eu (not opened; status unverified)
- EMD2: Directive 2009/110/EC. https://eur-lex.europa.eu/eli/dir/2009/110/oj (not opened)
- MiCA: Regulation (EU) 2023/1114. https://eur-lex.europa.eu/eli/reg/2023/1114/oj (not opened)
- CCD2: Directive (EU) 2023/2225. https://eur-lex.europa.eu/eli/dir/2023/2225/oj (not opened)
- FCA-PSR / FCA-CC: FCA payments and e-money guidance (Perimeter Guidance, PERG 15) and consumer credit pages. https://www.fca.org.uk (not opened)
- FSMA: Financial Services and Markets Act 2000 (Regulated Activities) Order. https://www.legislation.gov.uk (not opened)
- FINTRAC / BOC: https://fintrac-canafe.canada.ca ; https://www.bankofcanada.ca/core-functions/retail-payments-supervision/ (not opened)
- AUSTRAC / ASIC: https://www.austrac.gov.au ; https://asic.gov.au (not opened)
- FSPR: NZ Financial Service Providers Register. https://fspr.companiesoffice.govt.nz (not opened)
- HOWEY / SEC: SEC v. W.J. Howey Co., 328 U.S. 293 (1946); SEC Framework for "Investment Contract" Analysis of Digital Assets. https://www.sec.gov (not opened)
- UCTD / CRA / CRD / DMCCA: Directive 93/13/EEC; Consumer Rights Act 2015; Directive 2011/83/EU; Digital Markets, Competition and Consumers Act 2024. https://eur-lex.europa.eu ; https://www.legislation.gov.uk (not opened)
- DSA: Regulation (EU) 2022/2065. https://eur-lex.europa.eu/eli/reg/2022/2065/oj (not opened)
- GDPR / GDPR-8 / 15 / 17 / 33 / 35 / V: Regulation (EU) 2016/679. https://eur-lex.europa.eu/eli/reg/2016/679/oj (not opened)
- EDPB-CP: EDPB Guidelines 07/2020 on controller and processor concepts. https://www.edpb.europa.eu (not opened)
- ICO-CP: ICO guidance on controllers and processors. https://ico.org.uk/for-organisations/ (not opened)
- EDPB-BC: EDPB Guidelines 02/2025 on processing personal data through blockchain technologies, v2.0 adopted 7 July 2026. https://www.edpb.europa.eu/system/files/2026-07/edpb_guidelines_202502_blockchain_v2_en.pdf (search listing; commentary at https://www.mondaq.com/fin-tech/1617044/blockchain-and-gdpr-edpb-guidelines-022025-adopted)
- EDPB-DPIA: Article 29 WP248 / national DPIA lists. https://www.edpb.europa.eu (not opened)
- CCPA: California Privacy Protection Agency. https://cppa.ca.gov (not opened)
- COPPA: 16 CFR Part 312. https://www.ecfr.gov (not opened)
- AADC: ICO Age appropriate design code. https://ico.org.uk (not opened)
- OPC / OAIC: https://www.priv.gc.ca ; https://www.oaic.gov.au ; NZ OPC https://www.privacy.org.nz (not opened)
- EAA: Directive (EU) 2019/882. https://eur-lex.europa.eu/eli/dir/2019/882/oj (not opened)
- ADA-WEB: DOJ ADA web rule fact sheet (Title II, 2024). https://www.ada.gov (not opened)
- OFAC: https://ofac.treasury.gov (not opened)
