# Personal Android companion

This project uses Capacitor 8 to package the existing React app. It shares the
same login and Go API as the web app. The **Income & Expenses** screen provides a
separate personal ledger and a review inbox for supported bank/card SMS.

## Build an APK on Windows

Prerequisites: Node.js 22+, Android Studio, JDK 21, Android SDK platform 36 and
build tools required by the generated Gradle project. Configure Android Studio's
SDK location and Gradle JDK. Do not commit SDK paths, signing keys or passwords.

Deploy the backend behind HTTPS first. A phone cannot reach the development PC
through its own `localhost`. The backend CORS configuration accepts the packaged
Android origin `https://localhost` in addition to the configured web origin.
Keep production JWT/admin credentials configured appropriately.

From this directory in PowerShell:

```powershell
npm install
$env:VITE_API_URL = 'https://YOUR-BACKEND-HOST/api/v1'
npm run android:sync
npm run android:open
```

Replace the example URL with your actual backend. The Android build script rejects
a missing or non-HTTPS API URL; web development continues to use its existing
configuration. Build and run on a device from Android Studio, or after configuring
`JAVA_HOME` and the SDK location:

```powershell
Set-Location android
.\gradlew.bat assembleDebug
```

The debug APK will be in `android\app\build\outputs\apk\debug\app-debug.apk`.
Install only APKs you trust. A debug APK is for personal testing, not Play release.
Use Android Studio's signed APK workflow for release distribution and protect the
keystore. No signing secret is included in this repository.

## SMS workflow

1. Add family members, bank accounts and credit cards in the app. Enter the
   correct bank/issuer and last four digits, then assign each account or card to
   its owner(s) on **Family**.
2. Log in on each phone with the shared account. In **Income & Expenses**, choose
   who owns that device. This preference is stored on that device only, separately
   for each login; it is not synchronized to the other phone.
3. Choose where the scan starts: **From a date** (default: the 1st of the current
   month; up to 89 days back) or **From the last few days** (1-90). Read and
   accept the disclosure, then choose
   **Scan and queue for review**. Grant Android SMS permission. Denial leaves
   all manual features usable.
4. Supported alerts become pending records. A unique bank/last-four match links
   a saved account or card. A sole owner takes precedence over the device default;
   for a joint account the device member is suggested only if they are an owner.
   Ambiguous matches are left unlinked. Correct the account and member with
   **Review / edit**, then set status to **Confirmed**.
5. Review possible duplicate warnings. Mark duplicates or unwanted records **Rejected**.
   Confirmation with a warning requires explicit review; similar payments can be
   legitimate and are never rejected automatically. Rejected imports are retained
   to prevent the same SMS being re-added by a rescan.
6. Rescan to pick up new messages. This release has no background SMS listener.

Use the member and account filters for the shared transaction list and confirmed
totals. The per-member breakdown keeps unassigned records separate. Earlier
ledger records are not automatically changed; link and assign them when reviewing.
Both people using one login can see, edit and delete all household data. Family
profiles and the device preference are not separate accounts or access controls.

The scan is limited to 5,000 examined inbox messages / 500 matching bank messages
per run. A visible truncation notice requests a shorter scan window. Parser
coverage is conservative: issuer support does not mean every sender alias or SMS
template is understood. Unsupported messages are counted and not guessed.
Use manual entry for them, or paste a supported message to preview it.

SMS bodies are read temporarily on-device, parsed locally and not stored by the
app or uploaded to the API. The backend stores structured financial fields,
the SMS receipt time, last four digits, and a SHA-256 fingerprint. OTP/promo
templates are excluded.
Manual paste text stays in component memory until cleared, saved, or the screen
is closed. Never enter real OTPs or full account/card numbers.

Manage or revoke SMS access with **Manage SMS permission** or Android system app
settings. Removing permission stops future scans but does not remove previously
saved financial records.

## Accounting boundaries

- Integer paise storage avoids floating-point rounding on individual entries.
- Only confirmed income, expenses and expense refunds affect ledger totals.
- Transfers/card repayments and bill notices do not count as income or expenses.
- A generic bank credit is only an income *candidate*. Correct own-account
  transfers, investment redemptions, refunds and loan disbursements in review.
- Message receipt date is the default date; correct it to the actual transaction
  date when necessary. Numeric bill due dates are extracted only when explicit
  and valid; missing values remain blank.
- Rescanning the same sender/timestamp/body is idempotent, including request
  retries. Possible duplicates with the same day, type, amount and last four
  are highlighted when SMS receipt times are within five minutes. If a time is
  unavailable, same-day matches are shown as less certain. Different saved
  accounts and known different banks are not treated as the same account.
  Detection can miss duplicates and can flag legitimate repeated payments;
  inspect each warning. Credit-card module transactions are not matched.
- The ledger is separate from the existing Credit Cards and investment cashflow
  modules. It does not automatically settle statements, update card outstanding,
  alter net worth, or create loan/investment transactions.
- Bank & Cash monthly in/out totals include only confirmed ledger records linked
  to that account. Existing unlinked entries remain in overall ledger totals but
  are not attributed to an account until reviewed and linked.
- CSV download is exposed on the web app. Use web Settings for backup/restore;
  native file sharing/export is not implemented.
- No budgets, background reminders, automatic posting, offline queue or biometric
  lock yet. The packaged app currently inherits the web session storage; native
  keystore-backed credential storage is still a production-hardening item.

## Validation

```powershell
npm run test:sms
npm run build
npm run lint
Set-Location ..\backend
go test .\...
```

Parser fixtures are synthetic. Test the APK on a physical phone for permission
grant/denial/revocation, scan limits, supported real sender variants, rescan
deduplication, bill amounts, refunds, manual corrections and web/mobile sync.
Native compilation and device behavior cannot be inferred from a successful
TypeScript/Vite build.

Google Play submission needs a separate SMS-permission eligibility review. This
personal APK is not an approval claim or a method for bypassing device permission.
