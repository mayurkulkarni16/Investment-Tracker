# FinBoard

FinBoard (this repository: Investment Tracker) is a comprehensive personal finance platform to track investments, loans, and net worth in one place. Supports 10 investment/liability types with live market data, Monte Carlo projections, tax planning, portfolio insights, and financial goal tracking.

---

## Features

### Android Companion and Income/Expense Ledger
- Personal Android companion using the existing React app and Go API.
- Manual income, expense, refund, transfer and bill records with monthly totals,
  pending review, corrections, web CSV export and JSON backup/restore support.
- Link transactions to a saved bank account/card and family member; filter and
  see confirmed totals per member. Each phone can store its own SMS member
  default. Unique account/owner matches are suggested and remain editable.
- Local SMS parsing plus an opt-in Android inbox scan for initial HDFC, ICICI,
  Axis, SBI, OneCard and YES BANK templates. Imported records require review.
- Possible shared-account duplicates are flagged for review, not automatically
  rejected. Same-day matches without SMS timestamps are less certain.
- This is an initial slice: no background SMS ingestion or automatic statement
  reconciliation yet. The ledger is separate from investment cashflow/card balances.
- [Android setup and limitations](frontend/ANDROID.md) |
  [Complete mobile feature list and roadmap](MOBILE_FEATURES.md).

### Investment Tracking
- **Mutual Funds** — Add funds by AMFI scheme code with auto NAV fetching. CAS PDF import, ELSS lock-in tracking, SIP/purchase/redemption transactions, and 80C tax saving calculation.
- **Stocks** — Live prices via Yahoo Finance, auto-refreshed every 60s during market hours (Mon–Fri 9:15 AM – 3:30 PM IST). Buy/sell transactions, dividends, average cost, P&L, and day change.
- **Corporate Bonds** — Track coupon payments and principal repayments (bullet or staggered). Mark payouts as received, auto-generated interest schedules, and dynamic recalculation on principal reduction.
- **Fixed Deposits** — Cumulative and non-cumulative FDs with maturity amount calculation and auto-renewal tracking.
- **Provident Fund** — EPF, VPF, and PPF support. Monthly contribution tracking by financial year, employee + employer split, PF passbook PDF import, and interest calculation per EPFO rules.
- **NPS (National Pension System)** — Tier 1/Tier 2 accounts, contribution tracking, asset allocation (equity/corporate bond/govt securities/alternate), fund manager tracking, and 80CCD tax deduction computation.
- **Alternate Investments** — Operator-managed assets that pay a fixed monthly amount: EV fleet leasing (e.g. EZU Electric, Ridoji), cow ownership (e.g. Gir/Jersey at Amrutha Farms) and others. Auto-generated monthly payout schedule (fixed tenure or open-ended), manual payout confirmation with overdue alerts, buyback tracking, book value (straight-line to buyback, at cost, or manual), break-even months, milk-yield sanity check for livestock, quick-fill presets, duplicate-as-new (each investment keeps the rate it was agreed at), capital recovered, and XIRR.
- **SIPs** — Systematic investment plan tracking with installment recording, fund linking, frequency/date management, and completion stats.
- **Credit Cards** — Card management with billing statements, payment tracking, EMI conversions, transaction categorization, credit score history, and utilization monitoring.

### Household Finance & Planning
- **Bank & Cash** - Bank accounts, wallets and cash balances with dated balance history, emergency-fund flags and stale-balance warnings. Monthly in/out totals use confirmed ledger entries explicitly linked to each account; unlinked entries remain in overall ledger totals. Balances are recorded snapshots, not live bank feeds, and are not automatically reconciled.
- **Income, Expenses & Budgets** - Confirmed ledger totals, monthly savings rates, top spending categories and category limits with warnings at 80% and alerts at 100% or more. Pending SMS imports are excluded until reviewed.
- **Dashboard Money Overview** - Bank/cash totals, this month's income and expenses, budget alerts, life/health cover and premiums due within 30 days. Available independently of investment holdings.
- **Insurance** - Policy tracking, premium schedules, cover gaps and estimated 80C/80D premium figures. Scheduled premiums are estimates, not proof of payment or tax eligibility.
- **Gold & Real Estate** - Physical/digital gold, jewellery and property valuations, included in net worth.
- **Family** - Add family members with occupation (salaried, professional, business, pensioner, homemaker, student, minor…), date of birth, residency and declared tax regime. Assign holdings and liabilities to one member or split them between co-owners (e.g. a 60/40 joint home loan), record whose money funded them, and view member-level net worth.
- **Member income** - Monthly payslips per member and financial year (fixed pay, variable pay such as bonus/incentives/ESOPs, EPF, employer NPS, professional tax, TDS, rent paid), with remaining months projected from the latest payslip; plus pension, business/professional income (44AD/44ADA or actual), other income and deductions not tracked elsewhere.
- **FIRE Planner** - Retirement corpus, required savings, earliest feasible retirement age and simulated success probability. Assumptions and simulations are planning estimates, not guaranteed returns.
- **Document Vault** - Upload, download and delete supporting documents (10 MB per file), with expiry reminders. Document files are not included in JSON backups.
- **Broker CSV Import** - Preview Zerodha/Groww holdings and tradebook imports; duplicate trades are skipped on re-import.
- **Reminders** - In-app alerts plus optional email/Telegram delivery for premiums, policy expiry, budget overruns and expiring documents. Configure delivery channels in Settings.

### Loan Tracking
- **Home Loans** — Full lifecycle management: EMI recording, prepayments, floating rate changes, amortization schedule, under-construction support (tranche disbursements, pre-EMI interest), and auto-recalculation.
- **Personal Loans** — EMI recording, prepayments, rate changes, amortization schedule, and processing fee tracking.

### Analytics & Intelligence
- **Dashboard** — Portfolio summary with XIRR, asset allocation, risk metrics (equity/debt ratio, risk score, diversification, concentration risk, loan-to-asset ratio), upcoming payouts, loan and credit card summaries, and rebalancing suggestions.
- **Investment Projections** — Monte Carlo simulation (2,000 jointly simulated paths with market correlation) with P10/P50/P90 bands for 1–30 year horizons. Expected returns blend your XIRR with category long-term returns; FDs, PF, bonds and alternate investments follow their schedules. Bull/bear/base scenarios, optional "today's money" (inflation-adjusted) view, and a net-worth projection with loans amortised.
- **Cashflow Analysis** — Monthly inflow/outflow aggregation across all investment types with month-level drilldown.
- **Insights Engine** — Automated portfolio analysis generating actionable insights across risk, performance, tax, diversification, loan, goal, and opportunity categories with severity levels (critical/warning/info/positive).
- **Benchmark Comparison** — Nifty 50 and Sensex levels with 1Y/3Y/5Y returns (absolute and annualised) via Yahoo Finance, cached hourly; chart periods 1m–5y.
- **Tax Center** — Per-member old vs new regime comparison for FY 2025-26 and FY 2026-27 (Income-tax Act, 2025 section numbers shown alongside the old ones), using each member's income plus the interest, dividends, rent, capital gains and deductions (80C incl. PPF, 80CCD, 80D self/parents, 24(b), 80TTA/TTB) from the assets attributed to them. Applies clubbing (spouse, minor child) by default, with switches to turn either off and see tax as if each person were taxed on what is in their name (flagged as not how the law works), 87A rebate with marginal relief, surcharge, cess and senior-citizen slabs. Shows the recommended regime, savings, break-even deductions and household suggestions (regime switch, unused deductions, employer NPS, HRA, parents' 80D, joint home loan, LTCG harvesting, holding deposits with low-income members). A "Where to invest" advisor ranks family members by post-tax return for a new FD, debt/equity fund, ELSS, PPF, SSY, SCSS, NPS or bond. Estimates only — not tax advice.
- **Net Worth Tracking** — Real-time net worth (assets minus liabilities), daily snapshots via scheduler, and historical trend charts.
- **Financial Goals** — Set targets with amount/date (optionally inflation-adjusted), link investments with allocation % (max 100% per investment across goals), and see on-track status, projected completion date, shortfall and extra monthly investment needed based on your actual SIP/PF/NPS contributions.

### Platform Features
- **Authentication** — Email/password signup, JWT login, forgot/reset password, change password, admin role with user management and view-as capability.
- **Notifications** — Auto-generated alerts for upcoming events (EMIs, FD maturity, bond payouts, SIP dates, credit card due dates, goal deadlines) within 7 days. Mark read individually or all at once.
- **Data Export** — CSV export per module or all investments.
- **Backup & Restore** — JSON backup of investments, ledger, accounts, budgets, insurance, gold, property, profile assignments, member income records and reminder settings. Document files are excluded. Restore replaces only the current user's data; records are always re-assigned to the restoring user.
- **Profiles** — Multi-profile support with name, relationship, color and tax details (occupation, date of birth, residency, disability, spouse link, declared regime). PAN/Aadhaar are not stored.
- **Background Scheduler** — Runs daily at 20:00 IST (plus a catch-up shortly after startup) to refresh MF NAVs and stock prices, take each user's monthly net worth snapshot, and generate their notifications.

---

## Tech Stack

| Layer      | Technology                           |
|------------|--------------------------------------|
| Frontend   | React 19, TypeScript, Vite, Recharts |
| Backend    | Go 1.26, Gin                         |
| Database   | MongoDB                              |
| NAV Data   | AMFI API (mfapi.in)                  |
| Stock Data | Yahoo Finance API                    |
| Auth       | JWT + bcrypt                         |

---

## Prerequisites

- [Go 1.21+](https://golang.org/dl/)
- [Node.js 18+](https://nodejs.org/)
- [MongoDB](https://www.mongodb.com/try/download/community) running on `localhost:27017`

---

## Getting Started

### 1. Clone the repository

```bash
git clone <repo-url>
cd investment-tracker
```

### 2. Install root dependencies

```bash
npm install
```

### 3. Install frontend dependencies

```bash
cd frontend && npm install && cd ..
```

### 4. Run both servers together

```bash
npm start
```

This starts:
- **Backend** on `http://localhost:8080`
- **Frontend** on `http://localhost:5173`

### Run separately

```bash
# Backend
cd backend
go run main.go

# Frontend (in another terminal)
cd frontend
npm run dev
```

---

## Configuration

The backend reads configuration from environment variables with sensible defaults:

| Variable      | Default                     | Description              |
|---------------|-----------------------------|--------------------------|
| `MONGO_URI`   | `mongodb://localhost:27017` | MongoDB connection string |
| `DB_NAME`     | `investment_tracker`        | MongoDB database name    |
| `SERVER_PORT` | `8080`                      | Backend server port      |
| `JWT_SECRET`  | development default         | JWT signing secret; override in production |

Optional outbound messaging configuration:

| Variable | Default | Description |
|----------|---------|-------------|
| `SMTP_HOST` | unset | SMTP server for reminders and password-reset emails |
| `SMTP_PORT` | `587` | SMTP server port |
| `SMTP_USERNAME` | unset | SMTP authentication username |
| `SMTP_PASSWORD` | unset | SMTP authentication password |
| `SMTP_FROM` | unset | Sender email address |
| `TELEGRAM_BOT_TOKEN` | unset | Telegram bot token; each user sets their chat ID in Settings |

Without SMTP, password-reset links are logged by the server rather than emailed.
Keep credentials in deployment environment variables, not source control.
The authenticated `POST /api/v1/auth/refresh` endpoint renews the current JWT for companion clients.

Household-finance API groups (all under `/api/v1`): `/accounts`, `/ledger/summary`,
`/budgets`, `/insurance`, `/gold`, `/properties`, `/family`, `/fire`, `/documents`,
`/stocks/import-broker`, `/notifications/settings` and `/notifications/test`.
`GET /finance/overview` supplies the web Dashboard and mobile home screen with the
same household totals. CSV export also includes accounts, budgets, insurance, gold and property.

---

## Project Structure

```
investment-tracker/
├── backend/                   # Go REST API
│   ├── main.go                # Entry point, router setup, 103 routes
│   ├── config/                # Environment config
│   ├── database/              # MongoDB connection
│   ├── middleware/             # JWT auth middleware
│   ├── models/                # Data models + request/response types (18 files)
│   ├── repository/            # MongoDB CRUD layer (15 repos)
│   ├── services/              # Business logic, schedulers, fetchers (24 services)
│   └── handlers/              # HTTP handlers (22 handlers)
├── frontend/                  # React SPA
│   └── src/
│       ├── api/               # Axios API clients per resource
│       ├── components/        # Shared UI components (Sidebar, TopBar, CommandPalette, …)
│       ├── context/           # Auth context
│       ├── pages/             # 21 pages (4 auth + 17 app pages)
│       ├── types/             # TypeScript interfaces mirroring Go models
│       └── utils/             # CAS parser, PF parser, formatting helpers
└── package.json               # Root scripts (start both servers)
```

---

## API Reference

All endpoints are prefixed with `/api/v1`. Authenticated routes require `Authorization: Bearer <token>` header.

### Auth (Public)
| Method | Endpoint                    | Description                  |
|--------|-----------------------------|------------------------------|
| POST   | `/auth/signup`              | Register new user            |
| POST   | `/auth/login`               | Login, returns JWT           |
| POST   | `/auth/forgot-password`     | Send password reset token    |
| POST   | `/auth/reset-password`      | Reset password with token    |

### Auth (Authenticated)
| Method | Endpoint                    | Description                  |
|--------|-----------------------------|------------------------------|
| GET    | `/auth/me`                  | Get current user profile     |
| PUT    | `/auth/me`                  | Update profile               |
| PUT    | `/auth/change-password`     | Change password              |
| GET    | `/auth/users`               | List all users (admin only)  |

### Dashboard
| Method | Endpoint                    | Description                           |
|--------|-----------------------------|---------------------------------------|
| GET    | `/dashboard`                | Portfolio summary across all types     |
| POST   | `/dashboard/rebalance`      | Get rebalancing suggestions            |

### Mutual Funds
| Method | Endpoint                                    | Description                         |
|--------|---------------------------------------------|-------------------------------------|
| GET    | `/mutual-funds`                             | List all mutual funds               |
| POST   | `/mutual-funds`                             | Add a fund                          |
| GET    | `/mutual-funds/:id`                         | Get fund by ID                      |
| PUT    | `/mutual-funds/:id`                         | Update fund                         |
| DELETE | `/mutual-funds/:id`                         | Delete fund                         |
| POST   | `/mutual-funds/:id/transactions`            | Add SIP / purchase / redemption     |
| PUT    | `/mutual-funds/:id/transactions/:txnId`     | Update transaction                  |
| DELETE | `/mutual-funds/:id/transactions/:txnId`     | Delete transaction                  |
| POST   | `/mutual-funds/:id/refresh-nav`             | Refresh NAV for one fund            |
| POST   | `/mutual-funds/refresh-nav`                 | Refresh NAV for all funds           |
| POST   | `/mutual-funds/import`                      | Bulk import from CAS data           |
| POST   | `/mutual-funds/recalculate`                 | Recalculate all fund totals         |

### Stocks
| Method | Endpoint                                    | Description                         |
|--------|---------------------------------------------|-------------------------------------|
| GET    | `/stocks`                                   | List all stocks                     |
| POST   | `/stocks`                                   | Add a stock                         |
| GET    | `/stocks/:id`                               | Get stock by ID                     |
| PUT    | `/stocks/:id`                               | Update stock details                |
| DELETE | `/stocks/:id`                               | Delete stock                        |
| POST   | `/stocks/:id/transactions`                  | Add a buy/sell transaction          |
| PUT    | `/stocks/:id/transactions/:txnId`           | Update transaction                  |
| DELETE | `/stocks/:id/transactions/:txnId`           | Delete transaction                  |
| POST   | `/stocks/:id/dividends`                     | Add dividend record                 |
| DELETE | `/stocks/:id/dividends/:divId`              | Delete dividend                     |
| POST   | `/stocks/:id/refresh-price`                 | Refresh price for one stock         |
| POST   | `/stocks/refresh-prices`                    | Refresh prices for all stocks       |
| GET    | `/stocks/market-status`                     | Check if Indian market is open      |

### Corporate Bonds
| Method | Endpoint                                      | Description                    |
|--------|-----------------------------------------------|--------------------------------|
| GET    | `/corporate-bonds`                            | List all bonds                 |
| POST   | `/corporate-bonds`                            | Add a bond                     |
| GET    | `/corporate-bonds/:id`                        | Get bond by ID                 |
| PUT    | `/corporate-bonds/:id`                        | Update bond                    |
| DELETE | `/corporate-bonds/:id`                        | Delete bond                    |
| PUT    | `/corporate-bonds/:id/payouts/:payoutId`      | Mark a payout as received      |

### Fixed Deposits
| Method | Endpoint               | Description        |
|--------|------------------------|--------------------|
| GET    | `/fixed-deposits`      | List all FDs       |
| POST   | `/fixed-deposits`      | Add a FD           |
| GET    | `/fixed-deposits/:id`  | Get FD by ID       |
| PUT    | `/fixed-deposits/:id`  | Update FD          |
| DELETE | `/fixed-deposits/:id`  | Delete FD          |

### Provident Fund
| Method | Endpoint                          | Description                         |
|--------|-----------------------------------|-------------------------------------|
| GET    | `/provident-fund`                 | List PF accounts                    |
| POST   | `/provident-fund`                 | Create PF account                   |
| GET    | `/provident-fund/:id`             | Get PF account by ID                |
| PUT    | `/provident-fund/:id`             | Update PF account                   |
| DELETE | `/provident-fund/:id`             | Delete PF account                   |
| POST   | `/provident-fund/:id/entries`     | Add monthly contribution            |
| POST   | `/provident-fund/:id/import`      | Bulk import from PF passbook PDF    |

### Home Loans
| Method | Endpoint                              | Description                           |
|--------|---------------------------------------|---------------------------------------|
| GET    | `/home-loans`                         | List all home loans                   |
| POST   | `/home-loans`                         | Add a home loan                       |
| GET    | `/home-loans/:id`                     | Get loan by ID                        |
| PUT    | `/home-loans/:id`                     | Update loan                           |
| DELETE | `/home-loans/:id`                     | Delete loan                           |
| POST   | `/home-loans/:id/emi`                 | Record EMI payment                    |
| POST   | `/home-loans/:id/prepayment`          | Add prepayment                        |
| POST   | `/home-loans/:id/rate-change`         | Record interest rate change           |
| GET    | `/home-loans/:id/amortization`        | Get amortization schedule             |
| POST   | `/home-loans/:id/disbursement`        | Add tranche disbursement              |
| POST   | `/home-loans/:id/mark-complete`       | Mark construction complete            |
| POST   | `/home-loans/:id/recalculate`         | Recalculate loan                      |

### Personal Loans
| Method | Endpoint                              | Description                           |
|--------|---------------------------------------|---------------------------------------|
| GET    | `/personal-loans`                     | List all personal loans               |
| POST   | `/personal-loans`                     | Add a personal loan                   |
| GET    | `/personal-loans/:id`                 | Get loan by ID                        |
| PUT    | `/personal-loans/:id`                 | Update loan                           |
| DELETE | `/personal-loans/:id`                 | Delete loan                           |
| POST   | `/personal-loans/:id/emi`             | Record EMI payment                    |
| POST   | `/personal-loans/:id/prepayment`      | Add prepayment                        |
| POST   | `/personal-loans/:id/rate-change`     | Record interest rate change           |
| GET    | `/personal-loans/:id/amortization`    | Get amortization schedule             |

### NPS
| Method | Endpoint                        | Description                  |
|--------|---------------------------------|------------------------------|
| GET    | `/nps`                          | List NPS accounts            |
| POST   | `/nps`                          | Create NPS account           |
| GET    | `/nps/:id`                      | Get NPS account by ID        |
| PUT    | `/nps/:id`                      | Update NPS account           |
| DELETE | `/nps/:id`                      | Delete NPS account           |
| POST   | `/nps/:id/contributions`        | Add contribution             |

### Alternate Investments
`/vehicle-rentals/...` remains as an alias of every route below for older app builds.

| Method | Endpoint                                          | Description                                        |
|--------|---------------------------------------------------|----------------------------------------------------|
| GET    | `/alternate-investments`                          | List alternate investments                         |
| POST   | `/alternate-investments`                          | Create (generates monthly payout schedule; `tenure_months: 0` = open-ended) |
| GET    | `/alternate-investments/:id`                      | Get by ID                                          |
| PUT    | `/alternate-investments/:id`                      | Update terms (keeps recorded receipts)             |
| DELETE | `/alternate-investments/:id`                      | Delete                                             |
| PUT    | `/alternate-investments/:id/payouts/:payoutId`    | Mark payout received (date, amount) or pending     |
| POST   | `/alternate-investments/:id/mark-due-received`    | Mark all due payouts as received in full           |
| PUT    | `/alternate-investments/:id/buyback`              | Mark buyback received or pending                   |

### Credit Cards
| Method | Endpoint                              | Description                    |
|--------|---------------------------------------|--------------------------------|
| GET    | `/credit-cards`                       | List all cards                 |
| POST   | `/credit-cards`                       | Add a card                     |
| GET    | `/credit-cards/:id`                   | Get card by ID                 |
| PUT    | `/credit-cards/:id`                   | Update card                    |
| DELETE | `/credit-cards/:id`                   | Delete card                    |
| POST   | `/credit-cards/:id/statements`        | Add billing statement          |
| POST   | `/credit-cards/:id/pay-statement`     | Pay a statement                |
| POST   | `/credit-cards/:id/transactions`      | Add card transaction           |
| POST   | `/credit-cards/:id/emis`              | Add card EMI conversion        |
| POST   | `/credit-cards/:id/credit-score`      | Record credit score            |

### SIPs
| Method | Endpoint                        | Description                    |
|--------|---------------------------------|--------------------------------|
| GET    | `/sips`                         | List all SIPs                  |
| POST   | `/sips`                         | Create SIP plan                |
| GET    | `/sips/:id`                     | Get SIP by ID                  |
| PUT    | `/sips/:id`                     | Update SIP                     |
| DELETE | `/sips/:id`                     | Delete SIP                     |
| POST   | `/sips/:id/installments`        | Record SIP installment         |

### Goals
| Method | Endpoint                              | Description                    |
|--------|---------------------------------------|--------------------------------|
| GET    | `/goals`                              | List all goals                 |
| POST   | `/goals`                              | Create goal                    |
| GET    | `/goals/:id`                          | Get goal by ID                 |
| PUT    | `/goals/:id`                          | Update goal                    |
| DELETE | `/goals/:id`                          | Delete goal                    |
| POST   | `/goals/:id/link`                     | Link investment to goal        |
| POST   | `/goals/:id/link-batch`               | Batch link investments         |
| DELETE | `/goals/:id/link/:investmentId`       | Unlink investment from goal    |

### Analytics
| Method | Endpoint                  | Description                                     |
|--------|---------------------------|-------------------------------------------------|
| GET    | `/projections`            | Monte Carlo projections (P10/P50/P90); `years`, `scenario`, `inflation` |
| GET    | `/cashflow`               | Monthly cashflow summary                        |
| GET    | `/cashflow/:month`        | Cashflow drilldown for a month                  |
| GET    | `/insights`               | Portfolio insights and recommendations          |
| GET    | `/benchmarks`             | Benchmark comparison (Nifty 50, Sensex)         |
| GET    | `/tax/summary`            | Tax deduction summary (80C, 80D, 24b, 80CCD); `fy`, optional `profile_id` |
| GET    | `/tax/capital-gains`      | Capital gains breakdown; `fy`, optional `profile_id` |
| GET    | `/tax/meta`               | Supported FYs, current FY, occupations, placement instruments |
| GET    | `/tax/family`             | Per-member old vs new regime comparison and suggestions; `fy` |
| GET    | `/tax/settings`           | Household tax settings (clubbing switches)      |
| PUT    | `/tax/settings`           | Save `ignore_spouse_clubbing` / `ignore_minor_clubbing` |
| GET    | `/tax/placement`          | Rank members for a new investment; `amount`, `instrument`, `expected_return_pct`, `funded_by`, `fy` |
| GET    | `/tax/incomes`            | Member income records for a FY                  |
| GET    | `/tax/incomes/:profileId` | One member's income record (empty template if none) |
| PUT    | `/tax/incomes/:profileId` | Save monthly payslips, other income and deductions; `fy` |
| DELETE | `/tax/incomes/:profileId` | Delete a member's income record; `fy`           |

### Net Worth
| Method | Endpoint                  | Description                    |
|--------|---------------------------|--------------------------------|
| GET    | `/net-worth/current`      | Current net worth breakdown    |
| POST   | `/net-worth/snapshot`     | Take manual snapshot           |
| GET    | `/net-worth/history`      | Historical net worth trend     |

### Notifications
| Method | Endpoint                       | Description                    |
|--------|--------------------------------|--------------------------------|
| GET    | `/notifications`               | List all notifications         |
| GET    | `/notifications/unread`        | List unread notifications      |
| POST   | `/notifications/generate`      | Generate new notifications     |
| PUT    | `/notifications/:id/read`      | Mark notification as read      |
| PUT    | `/notifications/read-all`      | Mark all as read               |

### Profiles, Export & Backup
| Method | Endpoint                  | Description                    |
|--------|---------------------------|--------------------------------|
| GET    | `/profiles`               | List profiles                  |
| POST   | `/profiles`               | Create profile                 |
| PUT    | `/profiles/:id`           | Update profile                 |
| DELETE | `/profiles/:id`           | Delete profile                 |
| GET    | `/export/csv`             | Export investments as CSV      |
| GET    | `/backup`                 | Full JSON backup               |
| POST   | `/backup/restore`         | Restore from JSON backup       |

---

## MongoDB Collections

| Collection              | Description                                      |
|-------------------------|--------------------------------------------------|
| `users`                 | User accounts with auth credentials              |
| `stocks`                | Stock holdings and transactions                   |
| `mutual_funds`          | MF holdings with embedded transactions            |
| `corporate_bonds`       | Bond holdings with interest/principal schedules    |
| `fixed_deposits`        | FD records                                        |
| `provident_fund_entries`| PF accounts with yearly contribution entries      |
| `home_loans`            | Home loan records with EMI/prepayment history     |
| `personal_loans`        | Personal loan records with EMI history            |
| `nps_accounts`          | NPS accounts with contributions                   |
| `vehicle_rentals`       | Alternate investments (EV leasing, livestock, ...) with payouts; name kept from before the rename |
| `credit_cards`          | Credit cards with statements and EMIs             |
| `sips`                  | SIP plans with installment records                |
| `goals`                 | Financial goals with linked investments           |
| `net_worth_snapshots`   | Monthly net worth snapshots per user (updated daily) |
| `notifications`         | User notifications                                |
| `profiles`              | Family members with tax details (occupation, DOB, residency, regime) |
| `profile_assignments`   | Asset/liability ownership: member shares (%) and who funded it |
| `member_incomes`        | Per member, per FY: monthly payslips, other income, manual deductions, tax paid |
| `tax_settings`          | Household tax settings: clubbing switches for spouse / minor income |

---

## Guides

### Adding a Stock

1. Navigate to **Stocks** in the sidebar.
2. Click **+ Add Stock**, enter the company name, NSE/BSE symbol (e.g., `RELIANCE`, `TCS`, `INFY`), and exchange.
3. Add buy transactions with date, quantity, and price per share.
4. Click **Refresh Prices** to fetch live prices, or enable **Auto-refresh** for automatic updates every 60 seconds while the market is open.

### Importing Mutual Funds from CAS

1. Download your Consolidated Account Statement PDF from [CAMS](https://www.camsonline.com/) or [KFintech](https://www.kfintech.com/).
2. Go to **Mutual Funds** → **Import CAS**.
3. Upload the PDF (enter password if the file is protected).
4. Review the parsed funds and transactions, then click **Import**.
