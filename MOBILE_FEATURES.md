# Mobile App - Proposed Features

Status: initial implementation plus roadmap. The existing web app and Go API
remain the shared source of financial data. Chosen target: personal Android APK,
with initial sender/template support for HDFC, ICICI, Axis, SBI, OneCard and YES BANK.

## Implemented first slice

- Capacitor Android project reusing the existing web app and authenticated API.
- Shared manual income/expense/refund/transfer/bill ledger, monthly totals,
  review/edit/reject, web CSV export, and inclusion in JSON backup/restore.
- Local, conservative SMS parser; paste-to-preview on web and mobile.
- Android native inbox scanner with opt-in disclosure and runtime permission,
  bounded historical window, and structured pending imports.
- Atomic retry/rescan protection for the same sender, timestamp, and message;
  no raw SMS persistence/upload.
- All imports require review before affecting totals.

Not yet implemented: background SMS reception, automatic posting, statement/card
reconciliation, cross-source duplicate matching, budgets, push reminders,
biometrics, secure native credential storage, or offline synchronization.
The ledger deliberately does not modify existing investment cashflow, net worth,
or card balances. Native APK compilation/device verification requires Android
Studio, JDK 21 and Android SDK 36; see [setup](frontend/ANDROID.md).

The remaining sections are the proposed complete roadmap, not a claim that all
features below have shipped.

## 1. Platform and delivery

- Android-first is recommended when automatic bank SMS import is a priority.
- Reuse the existing React interface through Capacitor where practical, with
  mobile navigation and native integrations. Confirm this against a dedicated
  React Native implementation before development.
- Use the same account and backend as the web app; confirmed changes appear on
  both clients after synchronization.
- An installable web app is an alternative for mobile access, but cannot read the
  device SMS inbox or continuously receive bank SMS.
- iOS can support the finance features, but not unrestricted SMS inbox access.
  Provide paste/import and potentially a share extension for user-supplied text
  where the source app supports sharing.
- Android SMS access requires explicit user permission. Google Play distribution
  additionally requires a restricted-permission declaration and review; its
  SMS-based money-management exception is not a guarantee of approval.
- A personal Android APK avoids Play distribution review, but still requires
  device permission, clear disclosure, and compliance with Android restrictions.
- No bank credentials, payment initiation, or automatic bill payment in scope.

## 2. First release: mobile finance companion

- Login, logout, existing profiles, and shared web/mobile data.
- Mobile-friendly dashboard: investments, net worth, liabilities, and upcoming dues.
- Access existing investment modules: mutual funds, stocks, bonds, fixed deposits,
  provident fund, NPS, SIPs, and vehicle rentals.
- Access existing home-loan, personal-loan, credit-card, and goal workflows.
- Dedicated income-and-expense ledger, separate from investment cashflow.
- Manual transactions with amount, date, account, category, merchant, and notes.
- Bank-account records using a nickname and masked account identifier.
- Categories, filters, search, and monthly income/expense summaries.
- Review inbox for imported transaction candidates: edit, approve, reject, undo.
- Manual paste of a bank/card message as a fallback when SMS access is unavailable.
- Explicit loading, offline, synchronization-error, and permission-denied states.

## 3. Android bank SMS import

- Opt-in setup explaining which messages are processed and what is synchronized.
- User-selected historical import window with preview before confirmation.
- New-message detection when permitted by Android; show last successful scan and
  provide rescan because background delivery is not guaranteed.
- Parse supported debit, credit, UPI, IMPS, NEFT, ATM, fee, and interest templates.
- Extract amount, currency, transaction time when present, bank, masked account,
  merchant/payee, and transaction reference when available.
- Distinguish transaction amounts from balances, available limits, and bill totals.
- Treat unsupported or ambiguous templates as needing review, never as confirmed
  financial records. Start with tested templates for the user's banks.
- Categorization rules and remembered user corrections.
- Duplicate protection for repeated messages, rescans, retries, and multi-device
  sync; reconcile matching manual records without silently deleting them.
- Identify own-account transfers, refunds, reversals, and investment transfers.
  A credit is not necessarily income; a debit is not necessarily an expense.
- Default to approval before posting. Later, offer opt-in auto-posting only for
  supported, unambiguous templates and user-approved rules.
- Ignore OTPs, promotional texts, and unrelated personal messages.
- Parse locally by default; sync only necessary structured financial fields.
  Raw SMS retention/upload is disabled by default and requires separate consent.

## 4. Credit-card intelligence

- Detect supported purchase, refund, reversal, fee, interest, and payment messages.
- Match cards using bank plus masked identifier; ask if matching is ambiguous.
- Detect statement notifications with statement date, total due, minimum due, and
  due date only when explicitly present.
- Track unbilled spending separately from issued statements. Reconcile against
  statements rather than assuming SMS covers every transaction.
- Extract EMI notifications into reviewable candidates; do not invent missing
  tenure, interest, fees, or remaining installment information.
- Confirm bill payments against the corresponding statement.
- Avoid double counting: a card purchase is an expense; the bank debit and card
  credit for settling that purchase are linked repayment/transfer records.
- Show spending categories, utilization, and upcoming payment reminders.
- Mark SMS-derived balances and limits as observations with timestamps, not
  authoritative live bank balances.

## 5. Accounting and backend integration

- Add user-scoped accounts, ledger entries, import candidates, and provenance.
- Separate economic income/expenses from account movements and investment
  cashflow; label reports clearly.
- Use decimal-safe monetary storage, explicit currency, and consistent time zones.
- Make import confirmation and synchronization idempotent.
- Support pending, confirmed, rejected, reversed, and reconciled states.
- Link imports to existing card transactions, statements, loan payments, and
  investment records only after a reliable match or user confirmation.
- Extend web screens, summaries, CSV export, backup/restore, and deletion to cover
  the new ledger so mobile data is not isolated.
- Treat the existing card model as an integration starting point, not a complete
  ledger: it currently associates transactions with statements and needs explicit
  handling for unbilled transactions, refunds, and import provenance.

## 6. Additional features after the first release

- Monthly category budgets and overspending alerts.
- Recurring subscription detection and renewal reminders.
- Bill calendar for cards, utilities, rent, loans, insurance, and SIPs.
- Explainable unusual-spend and duplicate-charge alerts, not fraud guarantees.
- Statement CSV/PDF import and reconciliation for missing or unsupported SMS.
- Offline entry and queued sync with visible conflicts and retry controls.
- Biometric app lock, secure session storage, and notification privacy controls.
- Searchable merchant rules, split transactions, tags, and household profiles.
- Savings-rate trends, emergency-fund goals, and month-end summaries.
- Android home-screen widgets and quick-add shortcuts.
- Optional regulated account-data integrations, subject to provider availability,
  consent, commercial agreements, and separate feasibility work.

## 7. Privacy and reliability requirements

- Permission denial must not block manual finance features.
- Provide pause import, revoke access, clear local message data, and delete data.
- No SMS bodies, OTPs, full account numbers, or authentication tokens in logs.
- Use authenticated HTTPS for production sync and platform-secured local storage.
- Keep a reviewable source/matching history and reversible user actions.
- Test parsing with synthetic fixtures, including ambiguous amounts, Indian digit
  grouping, duplicate alerts, missing fields, refunds, and transfer pairs.
- Test end-to-end totals: imports must not inflate income, expenses, card dues,
  investment cashflow, or net worth.
- Validate permission and background behavior on a physical Android device.
- Store readiness and signed release builds are separate from local development;
  neither should be claimed without platform testing and required approvals.

## Suggested delivery order

1. Confirm Android/iOS scope, distribution, banks, and initial workflows.
2. Deliver mobile access plus shared manual ledger and web reporting.
3. Add paste-to-import, review, categorization, and reconciliation.
4. Add Android SMS permissions, historical scan, and new-message ingestion.
5. Add budgets, advanced alerts, offline sync, and release hardening.

## Platform references

- [Google Play SMS and Call Log permissions policy](https://support.google.com/googleplay/android-developer/answer/10208820?hl=en)
- [Apple SMS and Call Reporting documentation](https://developer.apple.com/documentation/identitylookup)
