# Investment Tracker - Architecture & Schema Design

## Tech Stack

| Layer      | Technology     | Purpose                        |
|------------|---------------|--------------------------------|
| Frontend   | React (Vite)  | SPA with dashboard & forms     |
| Backend    | Go (Gin)      | REST API, NAV auto-fetch, bond schedule calculator |
| Database   | MongoDB       | Document store for all investments |
| NAV Source | AMFI API      | Auto-fetch mutual fund NAVs    |

---

## System Architecture

```
┌─────────────────────────────────────────────────────┐
│                   React Frontend                     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ │
│  │Dashboard │ │ MF/ELSS  │ │  Bonds   │ │ FD/PF  │ │
│  │(Summary) │ │  Module  │ │  Module  │ │ Module │ │
│  └──────────┘ └──────────┘ └──────────┘ └────────┘ │
└───────────────────┬─────────────────────────────────┘
                    │ HTTP (localhost:8080)
┌───────────────────▼─────────────────────────────────┐
│                  Go Backend (Gin)                     │
│  ┌────────────────────────────────────────────────┐  │
│  │               REST API Layer                    │  │
│  │  /api/v1/mutual-funds                          │  │
│  │  /api/v1/corporate-bonds                       │  │
│  │  /api/v1/fixed-deposits                        │  │
│  │  /api/v1/provident-fund                        │  │
│  │  /api/v1/dashboard                             │  │
│  └────────────────────────────────────────────────┘  │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│  │  NAV Fetcher │ │ Bond Schedule│ │   Interest   │ │
│  │  (AMFI API)  │ │  Calculator  │ │  Calculator  │ │
│  └──────────────┘ └──────────────┘ └──────────────┘ │
└───────────────────┬─────────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────────┐
│              MongoDB (localhost:27017)                │
│  Database: investment_tracker                        │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│  │ mutual_funds │ │corporate_    │ │fixed_deposits│ │
│  │              │ │bonds         │ │              │ │
│  ├──────────────┤ ├──────────────┤ ├──────────────┤ │
│  │provident_    │ │bond_payouts  │ │  categories  │ │
│  │fund_entries  │ │              │ │              │ │
│  └──────────────┘ └──────────────┘ └──────────────┘ │
└─────────────────────────────────────────────────────┘
```

---

## API Endpoints

### Dashboard
| Method | Endpoint               | Description                     |
|--------|------------------------|---------------------------------|
| GET    | /api/v1/dashboard      | Portfolio summary across all types |

### Mutual Funds (incl. ELSS)
| Method | Endpoint                          | Description                      |
|--------|-----------------------------------|----------------------------------|
| GET    | /api/v1/mutual-funds              | List all MF investments          |
| POST   | /api/v1/mutual-funds              | Add new MF investment            |
| GET    | /api/v1/mutual-funds/:id          | Get single MF details            |
| PUT    | /api/v1/mutual-funds/:id          | Update MF investment             |
| DELETE | /api/v1/mutual-funds/:id          | Delete MF investment             |
| POST   | /api/v1/mutual-funds/:id/transactions | Add SIP/additional purchase  |
| GET    | /api/v1/mutual-funds/refresh-nav  | Trigger NAV refresh for all funds|

### Corporate Bonds
| Method | Endpoint                              | Description                      |
|--------|---------------------------------------|----------------------------------|
| GET    | /api/v1/corporate-bonds               | List all bonds                   |
| POST   | /api/v1/corporate-bonds               | Add new bond                     |
| GET    | /api/v1/corporate-bonds/:id           | Get single bond with schedules   |
| PUT    | /api/v1/corporate-bonds/:id           | Update bond                      |
| DELETE | /api/v1/corporate-bonds/:id           | Delete bond                      |
| PUT    | /api/v1/corporate-bonds/:id/payouts/:payoutId | Mark a payout as received |
| GET    | /api/v1/corporate-bonds/:id/schedule  | Get full payout schedule         |

### Fixed Deposits
| Method | Endpoint                     | Description                      |
|--------|------------------------------|----------------------------------|
| GET    | /api/v1/fixed-deposits       | List all FDs                     |
| POST   | /api/v1/fixed-deposits       | Add new FD                       |
| GET    | /api/v1/fixed-deposits/:id   | Get single FD                    |
| PUT    | /api/v1/fixed-deposits/:id   | Update FD                        |
| DELETE | /api/v1/fixed-deposits/:id   | Delete FD                        |

### Provident Fund
| Method | Endpoint                          | Description                      |
|--------|-----------------------------------|----------------------------------|
| GET    | /api/v1/provident-fund            | Get PF summary                   |
| POST   | /api/v1/provident-fund            | Add/Create PF account            |
| POST   | /api/v1/provident-fund/:id/entries| Add monthly contribution entry   |
| GET    | /api/v1/provident-fund/:id/entries| Get all contribution entries     |
| PUT    | /api/v1/provident-fund/:id        | Update PF details                |

---

## MongoDB Schemas

### 1. Collection: `mutual_funds`

Tracks each mutual fund holding. Each transaction (SIP purchase, lumpsum, redemption) is stored inside.

```json
{
  "_id": "ObjectId",
  "fund_name": "Axis Long Term Equity Fund",
  "amc": "Axis Mutual Fund",
  "fund_type": "ELSS",               // enum: Equity, Debt, Hybrid, ELSS, Index, Liquid
  "scheme_code": "112323",            // AMFI scheme code for auto NAV fetch
  "folio_number": "1234567890",
  "is_elss": true,                    // quick filter for ELSS
  "lock_in_end_date": "2029-04-15",   // ELSS: 3-year lock-in from each purchase
  "transactions": [
    {
      "transaction_id": "UUID",
      "date": "2026-04-15",
      "type": "purchase",             // enum: purchase, sip, redemption, switch_in, switch_out
      "amount": 50000.00,
      "nav_at_purchase": 45.23,
      "units": 1105.46,
      "lock_in_end": "2029-04-15"     // per-transaction lock-in for ELSS
    }
  ],
  "total_units": 1105.46,
  "total_invested": 50000.00,
  "current_nav": 48.50,              // auto-fetched
  "current_value": 53614.81,         // total_units * current_nav
  "nav_last_updated": "2026-04-19T10:00:00Z",
  "gain_loss": 3614.81,
  "gain_loss_percent": 7.23,
  "notes": "",
  "created_at": "2026-04-15T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

**NAV Auto-Fetch**: Uses AMFI's public API (`https://api.mfapi.in/mf/{scheme_code}`) to fetch latest NAV by scheme code. NAV at purchase is entered manually for existing investments.

---

### 2. Collection: `corporate_bonds`

Complex schema to handle different payout frequencies and maturity types (bullet vs staggered).

```json
{
  "_id": "ObjectId",
  "bond_name": "Muthoot Finance NCD 2027",
  "issuer": "Muthoot Finance Ltd",
  "purchase_date": "2025-06-01",
  "investment_amount": 100000.00,       // face value / principal invested
  "coupon_rate": 9.50,                  // annual interest rate %
  "interest_payout_frequency": "quarterly",  // enum: monthly, quarterly, biannually, annually
  "maturity_date": "2027-06-01",
  "maturity_type": "staggered",         // enum: bullet, staggered

  // ───── PRINCIPAL REPAYMENT SCHEDULE ─────
  // For bullet: single entry at maturity_date for full amount
  // For staggered: multiple entries as per bond terms
  "principal_repayments": [
    {
      "repayment_id": "UUID-1",
      "scheduled_date": "2026-10-01",
      "amount": 33333.33,
      "status": "pending",              // enum: pending, received
      "received_date": null
    },
    {
      "repayment_id": "UUID-2",
      "scheduled_date": "2027-01-01",
      "amount": 33333.33,
      "status": "pending",
      "received_date": null
    },
    {
      "repayment_id": "UUID-3",
      "scheduled_date": "2027-06-01",
      "amount": 33334.34,
      "status": "pending",
      "received_date": null
    }
  ],

  // ───── INTEREST PAYOUT SCHEDULE ─────
  // Auto-generated based on frequency, start date, maturity date
  // Amount recalculated when principal reduces (staggered bonds)
  "interest_payouts": [
    {
      "payout_id": "UUID-A",
      "scheduled_date": "2025-09-01",
      "principal_at_time": 100000.00,   // principal on which interest is calculated
      "amount": 2375.00,               // (100000 * 9.5%) / 4 quarters
      "status": "received",
      "received_date": "2025-09-03"
    },
    {
      "payout_id": "UUID-B",
      "scheduled_date": "2025-12-01",
      "principal_at_time": 100000.00,
      "amount": 2375.00,
      "status": "received",
      "received_date": "2025-12-02"
    },
    // ... after first principal repayment of 33,333.33 on 2026-10-01
    {
      "payout_id": "UUID-X",
      "scheduled_date": "2027-01-01",
      "principal_at_time": 66666.67,    // remaining after first repayment
      "amount": 1583.33,               // (66666.67 * 9.5%) / 4
      "status": "pending",
      "received_date": null
    }
    // ... continues with decreasing principal
  ],

  "remaining_principal": 100000.00,     // updated as repayments are received
  "total_interest_earned": 4750.00,     // sum of received interest payouts
  "total_principal_returned": 0.00,     // sum of received principal repayments
  "status": "active",                   // enum: active, matured, partially_matured
  "notes": "",
  "created_at": "2025-06-01T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

**How Staggered Bond Interest Works (Your Example)**:
```
Bond: ₹1,00,000 | 2 years | Quarterly Interest | Staggered (last 3 quarters)

Year 1:
  Q1: Interest on ₹1,00,000
  Q2: Interest on ₹1,00,000
  Q3: Interest on ₹1,00,000
  Q4: Interest on ₹1,00,000

Year 2:
  Q1: Interest on ₹1,00,000 (no principal repayment yet)
  Q2: Interest on ₹1,00,000 + Principal ₹33,333 returned → remaining = ₹66,667
  Q3: Interest on ₹66,667   + Principal ₹33,333 returned → remaining = ₹33,334
  Q4: Interest on ₹33,334   + Principal ₹33,334 returned → remaining = ₹0 (matured)
```

---

### 3. Collection: `fixed_deposits`

```json
{
  "_id": "ObjectId",
  "bank_name": "SBI",
  "fd_number": "FD-2024-001",
  "principal_amount": 200000.00,
  "interest_rate": 7.10,               // annual %
  "start_date": "2025-01-15",
  "maturity_date": "2026-01-15",
  "tenure_months": 12,
  "interest_type": "cumulative",        // enum: cumulative, non_cumulative
  "payout_frequency": null,             // for non_cumulative: monthly, quarterly, annually
  "maturity_amount": 214200.00,         // for cumulative FDs
  "interest_earned": 14200.00,
  "is_auto_renewed": false,
  "status": "active",                   // enum: active, matured, premature_closed
  "notes": "",
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

---

### 4. Collection: `provident_fund_entries`

```json
{
  "_id": "ObjectId",
  "account_type": "EPF",               // enum: EPF, VPF, PPF
  "account_number": "MH/PUN/12345/67890",
  "employer_name": "TCS",              // for EPF
  "interest_rate": 8.25,               // current FY rate
  "financial_year_entries": [
    {
      "financial_year": "2025-2026",
      "monthly_contributions": [
        {
          "month": "2025-04",
          "employee_contribution": 1800.00,
          "employer_contribution": 1800.00,
          "total": 3600.00
        }
        // ... 12 months
      ],
      "opening_balance": 150000.00,
      "interest_earned": 12375.00,
      "closing_balance": 205575.00
    }
  ],
  "current_balance": 205575.00,
  "total_employee_contribution": 100000.00,
  "total_employer_contribution": 100000.00,
  "total_interest_earned": 5575.00,
  "notes": "",
  "created_at": "2024-01-01T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

---

## Project Structure

```
my-investments/
├── backend/                         # Go backend
│   ├── main.go                      # Entry point, server setup
│   ├── go.mod
│   ├── go.sum
│   ├── config/
│   │   └── config.go                # MongoDB URI, port, AMFI API config
│   ├── models/
│   │   ├── mutual_fund.go
│   │   ├── corporate_bond.go
│   │   ├── fixed_deposit.go
│   │   └── provident_fund.go
│   ├── handlers/
│   │   ├── mutual_fund_handler.go
│   │   ├── corporate_bond_handler.go
│   │   ├── fixed_deposit_handler.go
│   │   ├── provident_fund_handler.go
│   │   └── dashboard_handler.go
│   ├── services/
│   │   ├── mutual_fund_service.go
│   │   ├── corporate_bond_service.go # schedule generator, interest calc
│   │   ├── fixed_deposit_service.go
│   │   ├── provident_fund_service.go
│   │   └── nav_fetcher.go           # AMFI NAV auto-fetch
│   ├── repository/
│   │   ├── mutual_fund_repo.go
│   │   ├── corporate_bond_repo.go
│   │   ├── fixed_deposit_repo.go
│   │   └── provident_fund_repo.go
│   └── database/
│       └── mongodb.go               # Connection setup
│
├── frontend/                        # React frontend (Vite)
│   ├── package.json
│   ├── vite.config.ts
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── api/                     # API client functions
│   │   │   ├── mutualFunds.ts
│   │   │   ├── corporateBonds.ts
│   │   │   ├── fixedDeposits.ts
│   │   │   └── providentFund.ts
│   │   ├── components/
│   │   │   ├── Dashboard/
│   │   │   ├── MutualFunds/
│   │   │   ├── CorporateBonds/
│   │   │   ├── FixedDeposits/
│   │   │   ├── ProvidentFund/
│   │   │   └── common/              # Shared UI components
│   │   ├── pages/
│   │   │   ├── DashboardPage.tsx
│   │   │   ├── MutualFundsPage.tsx
│   │   │   ├── CorporateBondsPage.tsx
│   │   │   ├── FixedDepositsPage.tsx
│   │   │   └── ProvidentFundPage.tsx
│   │   └── types/                   # TypeScript interfaces
│   │       └── index.ts
│   └── index.html
│
└── ARCHITECTURE.md                  # This file
```

---

## Key Business Logic

### 1. Corporate Bond Schedule Generator
When a bond is added, the backend auto-generates:
- **Interest payout schedule**: Based on `interest_payout_frequency`, from `purchase_date` to `maturity_date`
- **Principal repayment schedule**: User enters this manually (staggered dates + amounts) or system sets single bullet repayment at maturity

When a principal repayment is marked as received:
1. `remaining_principal` decreases
2. All **future** interest payouts are recalculated using the new `remaining_principal`
3. `principal_at_time` on future payouts is updated

### 2. Mutual Fund NAV Auto-Fetch
- Uses free AMFI API: `https://api.mfapi.in/mf/{scheme_code}`
- On demand (manual refresh) or periodic (configurable)
- Updates `current_nav`, `current_value`, `gain_loss`, `gain_loss_percent`
- At data entry time: user manually enters `nav_at_purchase` for existing holdings

### 3. ELSS Lock-in Tracking
- Each ELSS transaction has its own `lock_in_end` (purchase_date + 3 years)
- Dashboard shows which units are locked vs unlocked
- Filter view for ELSS-only investments

### 4. FD Maturity Calculation
- Cumulative: compound interest calculation for maturity amount
- Non-cumulative: simple interest per payout period

### 5. PF Interest Calculation
- Monthly running balance method (as per EPFO rules)
- Interest calculated on monthly closing balances
- Credited annually

---

## Dashboard Summary

The dashboard aggregates across all investment types:

| Metric                     | Source                                    |
|----------------------------|-------------------------------------------|
| Total Invested             | Sum across all types                      |
| Current Value              | MF (units×NAV) + FD (principal/maturity) + Bonds (remaining principal) + PF (balance) |
| Total Gains                | Current Value - Total Invested            |
| Overall Return %           | (Gains / Invested) × 100                  |
| Monthly Interest Income    | Bonds interest + FD interest (non-cumulative) |
| Upcoming Payouts           | Next 30 days: bond interest, FD maturity, bond principal |
| ELSS Tax Saving (80C)      | Sum of ELSS investments in current FY (max ₹1.5L) |
| Asset Allocation Pie Chart | % in MF, Bonds, FD, PF                   |

---

## Data Flow: Adding a Corporate Bond (Staggered)

```
User fills form:
  → Bond name, issuer, amount, coupon rate
  → Interest frequency: quarterly
  → Maturity type: staggered
  → Principal repayment schedule:
      [{ date: "2027-01-01", amount: 33333 },
       { date: "2027-04-01", amount: 33333 },
       { date: "2027-07-01", amount: 33334 }]

Backend processes:
  1. Save bond document
  2. Generate interest payout schedule:
     - From purchase_date to first principal repayment: interest on full amount
     - Between repayments: interest on reduced principal
     - After last repayment: no more interest
  3. Return complete bond with all schedules

Frontend displays:
  - Bond summary card
  - Timeline of upcoming payouts (interest + principal)
  - Total expected returns
```

---

## Currency & Locale

- All amounts in INR (₹)
- Date format: DD-MM-YYYY (Indian standard)
- Financial year: April to March
