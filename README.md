# Investment Tracker

A personal finance dashboard to track all your investments in one place — mutual funds, stocks, corporate bonds, fixed deposits, and provident fund — with live stock prices during market hours and automated NAV fetching for mutual funds.

---

## Features

- **Stocks** — Live prices via Yahoo Finance, auto-refreshed every 60s during market hours (Mon–Fri 9:15 AM – 3:30 PM IST). Buy/sell transaction tracking, average cost, P&L, and day change.
- **Mutual Funds** — Add funds by AMFI scheme code and auto-fetch latest NAV. Import directly from a CAS (Consolidated Account Statement) PDF. ELSS lock-in tracking and 80C tax saving calculation.
- **Corporate Bonds** — Track coupon payments and principal repayments (bullet or staggered). Mark payouts as received and monitor upcoming cashflows.
- **Fixed Deposits** — Cumulative and non-cumulative FDs with maturity amount calculation and auto-renewal tracking.
- **Provident Fund** — EPF, VPF, and PPF support. Monthly contribution tracking by financial year, employee + employer split, and interest earned.
- **Dashboard** — Portfolio-wide summary with total invested, current value, total gains, asset allocation pie chart, investment overview charts, and upcoming payouts in the next 30 days.

---

## Tech Stack

| Layer     | Technology                          |
|-----------|-------------------------------------|
| Frontend  | React 19, TypeScript, Vite, Recharts |
| Backend   | Go 1.26, Gin                        |
| Database  | MongoDB                             |
| NAV Data  | AMFI API (mfapi.in)                 |
| Stock Data| Yahoo Finance API                   |

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

---

## Project Structure

```
investment-tracker/
├── backend/                   # Go REST API
│   ├── main.go                # Entry point, router setup
│   ├── config/                # Environment config
│   ├── database/              # MongoDB connection
│   ├── models/                # Data models + request/response types
│   ├── repository/            # MongoDB CRUD layer
│   ├── services/              # Business logic, NAV + price fetchers
│   └── handlers/              # HTTP handlers (Gin)
├── frontend/                  # React SPA
│   └── src/
│       ├── api/               # Axios API clients per resource
│       ├── components/        # Shared UI components (Sidebar)
│       ├── pages/             # One page per investment type + Dashboard
│       ├── types/             # TypeScript interfaces mirroring Go models
│       └── utils/             # CAS parser, PF parser, formatting helpers
└── package.json               # Root scripts (start both servers)
```

---

## API Reference

All endpoints are prefixed with `/api/v1`.

### Dashboard
| Method | Endpoint         | Description                           |
|--------|-----------------|---------------------------------------|
| GET    | `/dashboard`    | Portfolio summary across all types    |

### Stocks
| Method | Endpoint                          | Description                        |
|--------|-----------------------------------|------------------------------------|
| GET    | `/stocks`                         | List all stocks                    |
| POST   | `/stocks`                         | Add a stock                        |
| GET    | `/stocks/:id`                     | Get stock by ID                    |
| PUT    | `/stocks/:id`                     | Update stock details               |
| DELETE | `/stocks/:id`                     | Delete stock                       |
| POST   | `/stocks/:id/transactions`        | Add a buy/sell transaction         |
| POST   | `/stocks/:id/refresh-price`       | Refresh price for one stock        |
| POST   | `/stocks/refresh-prices`          | Refresh prices for all stocks      |
| GET    | `/stocks/market-status`           | Check if Indian market is open     |

### Mutual Funds
| Method | Endpoint                              | Description                       |
|--------|---------------------------------------|-----------------------------------|
| GET    | `/mutual-funds`                       | List all mutual funds             |
| POST   | `/mutual-funds`                       | Add a fund                        |
| GET    | `/mutual-funds/:id`                   | Get fund by ID                    |
| PUT    | `/mutual-funds/:id`                   | Update fund                       |
| DELETE | `/mutual-funds/:id`                   | Delete fund                       |
| POST   | `/mutual-funds/:id/transactions`      | Add SIP / purchase / redemption   |
| POST   | `/mutual-funds/:id/refresh-nav`       | Refresh NAV for one fund          |
| POST   | `/mutual-funds/refresh-nav`           | Refresh NAV for all funds         |
| POST   | `/mutual-funds/import`                | Bulk import from CAS data         |
| POST   | `/mutual-funds/recalculate`           | Recalculate all fund totals       |

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

---

## MongoDB Collections

| Collection              | Description                                      |
|-------------------------|--------------------------------------------------|
| `stocks`                | Stock holdings and transactions                  |
| `mutual_funds`          | MF holdings with embedded transactions           |
| `corporate_bonds`       | Bond holdings with interest and principal schedules |
| `fixed_deposits`        | FD records                                       |
| `provident_fund_entries`| PF accounts with yearly contribution entries     |

---

## Adding a Stock

1. Navigate to **Stocks** in the sidebar.
2. Click **+ Add Stock**, enter the company name, NSE/BSE symbol (e.g., `RELIANCE`, `TCS`, `INFY`), and exchange.
3. Add buy transactions with date, quantity, and price per share.
4. Click **Refresh Prices** to fetch live prices, or enable **Auto-refresh** for automatic updates every 60 seconds while the market is open.

## Importing Mutual Funds from CAS

1. Download your Consolidated Account Statement PDF from [CAMS](https://www.camsonline.com/) or [KFintech](https://www.kfintech.com/).
2. Go to **Mutual Funds** → **Import CAS**.
3. Upload the PDF (enter password if the file is protected).
4. Review the parsed funds and transactions, then click **Import**.

