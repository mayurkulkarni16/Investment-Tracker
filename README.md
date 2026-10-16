# Investment Tracker

A comprehensive personal finance platform to track investments, loans, and net worth in one place. Supports 10 investment/liability types with live market data, Monte Carlo projections, tax planning, portfolio insights, and financial goal tracking.

---

## Features

### Investment Tracking
- **Mutual Funds** — Add funds by AMFI scheme code with auto NAV fetching. CAS PDF import, ELSS lock-in tracking, SIP/purchase/redemption transactions, and 80C tax saving calculation.
- **Stocks** — Live prices via Yahoo Finance, auto-refreshed every 60s during market hours (Mon–Fri 9:15 AM – 3:30 PM IST). Buy/sell transactions, dividends, average cost, P&L, and day change.
- **Corporate Bonds** — Track coupon payments and principal repayments (bullet or staggered). Mark payouts as received, auto-generated interest schedules, and dynamic recalculation on principal reduction.
- **Fixed Deposits** — Cumulative and non-cumulative FDs with maturity amount calculation and auto-renewal tracking.
- **Provident Fund** — EPF, VPF, and PPF support. Monthly contribution tracking by financial year, employee + employer split, PF passbook PDF import, and interest calculation per EPFO rules.
- **NPS (National Pension System)** — Tier 1/Tier 2 accounts, contribution tracking, asset allocation (equity/corporate bond/govt securities/alternate), fund manager tracking, and 80CCD tax deduction computation.
- **SIPs** — Systematic investment plan tracking with installment recording, fund linking, frequency/date management, and completion stats.
- **Credit Cards** — Card management with billing statements, payment tracking, EMI conversions, transaction categorization, credit score history, and utilization monitoring.

### Loan Tracking
- **Home Loans** — Full lifecycle management: EMI recording, prepayments, floating rate changes, amortization schedule, under-construction support (tranche disbursements, pre-EMI interest), and auto-recalculation.
- **Personal Loans** — EMI recording, prepayments, rate changes, amortization schedule, and processing fee tracking.

### Analytics & Intelligence
- **Dashboard** — Portfolio summary with XIRR, asset allocation, risk metrics (equity/debt ratio, risk score, diversification, concentration risk, loan-to-asset ratio), upcoming payouts, loan and credit card summaries, and rebalancing suggestions.
- **Investment Projections** — Monte Carlo simulation (2,000 paths) with P10/P50/P90 confidence bands for 1–30 year horizons. Uses historical volatility per investment with bull/bear/base scenarios.
- **Cashflow Analysis** — Monthly inflow/outflow aggregation across all investment types with month-level drilldown.
- **Insights Engine** — Automated portfolio analysis generating actionable insights across risk, performance, tax, diversification, loan, goal, and opportunity categories with severity levels (critical/warning/info/positive).
- **Benchmark Comparison** — Compare portfolio performance against Nifty 50 and Sensex over 1m–10y periods via Yahoo Finance.
- **Tax Center** — Section 80C (EPF, ELSS, home loan principal, PPF, NPS), Section 24b (home loan interest), Section 80CCD (NPS additional deduction), and capital gains tracking.
- **Net Worth Tracking** — Real-time net worth (assets minus liabilities), daily snapshots via scheduler, and historical trend charts.
- **Financial Goals** — Set targets with amount/date, link investments with allocation %, track progress, projected completion date, shortfall analysis, and batch linking.

### Platform Features
- **Authentication** — Email/password signup, JWT login, forgot/reset password, change password, admin role with user management and view-as capability.
- **Notifications** — Auto-generated alerts for upcoming events (EMIs, FD maturity, bond payouts, SIP dates, credit card due dates, goal deadlines) within 7 days. Mark read individually or all at once.
- **Data Export** — CSV export per module or all investments.
- **Backup & Restore** — Full JSON backup of all user data (12 collections) with one-click restore.
- **Profiles** — Multi-profile support with name, relationship, and color coding.
- **Background Scheduler** — Runs daily at 20:00 IST to refresh MF NAVs, stock prices, and take net worth snapshots.

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
| `JWT_SECRET`  | (auto-generated)            | JWT signing secret       |

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
│       ├── components/        # Shared UI components (Sidebar)
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
| GET    | `/projections`            | Monte Carlo projections (P10/P50/P90)           |
| GET    | `/cashflow`               | Monthly cashflow summary                        |
| GET    | `/cashflow/:month`        | Cashflow drilldown for a month                  |
| GET    | `/insights`               | Portfolio insights and recommendations          |
| GET    | `/benchmarks`             | Benchmark comparison (Nifty 50, Sensex)         |
| GET    | `/tax/summary`            | Tax deduction summary (80C, 24b, 80CCD)         |
| GET    | `/tax/capital-gains`      | Capital gains breakdown                         |

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
| `credit_cards`          | Credit cards with statements and EMIs             |
| `sips`                  | SIP plans with installment records                |
| `goals`                 | Financial goals with linked investments           |
| `net_worth_snapshots`   | Daily net worth snapshots                         |
| `notifications`         | User notifications                                |
| `profiles`              | User profiles for multi-profile support           |

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

