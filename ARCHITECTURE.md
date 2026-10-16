# Investment Tracker - Architecture & Schema Design

## Tech Stack

| Layer        | Technology        | Purpose                                                        |
|--------------|-------------------|----------------------------------------------------------------|
| Frontend     | React 19 (Vite)   | SPA with 21 pages, Recharts visualizations                     |
| Backend      | Go (Gin)          | REST API (103 routes), schedulers, NAV/price fetchers          |
| Database     | MongoDB           | Document store — 15 collections                                |
| Auth         | JWT + bcrypt      | Stateless auth with role-based access (user/admin)             |
| NAV Source   | AMFI API          | Auto-fetch mutual fund NAVs                                    |
| Stock Source | Yahoo Finance API | Live stock prices + benchmark indices (Nifty 50, Sensex)       |

---

## System Architecture

```
┌───────────────────────────────────────────────────────────────────────────────┐
│                              React Frontend                                   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ ┌───────┐ ┌───────────┐  │
│  │Dashboard │ │ MF/ELSS  │ │  Bonds   │ │ FD/PF  │ │Stocks │ │ NPS/SIPs  │  │
│  │(Summary) │ │  Module  │ │  Module  │ │ Module │ │Module │ │  Module   │  │
│  └──────────┘ └──────────┘ └──────────┘ └────────┘ └───────┘ └───────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ ┌───────┐ ┌───────────┐  │
│  │  Home    │ │ Personal │ │ Credit   │ │ Goals  │ │  Tax  │ │Net Worth  │  │
│  │  Loans   │ │  Loans   │ │  Cards   │ │        │ │Center │ │           │  │
│  └──────────┘ └──────────┘ └──────────┘ └────────┘ └───────┘ └───────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────────────┐ ┌───────────┐   │
│  │Projec-   │ │ Cashflow │ │ Insights │ │   Settings      │ │   Auth    │   │
│  │tions(MC) │ │ Analysis │ │  Engine  │ │(Backup/Profiles)│ │ (JWT)     │   │
│  └──────────┘ └──────────┘ └──────────┘ └─────────────────┘ └───────────┘   │
└───────────────────────────┬───────────────────────────────────────────────────┘
                            │ HTTP (localhost:8080)
                            │ Authorization: Bearer <JWT>
┌───────────────────────────▼───────────────────────────────────────────────────┐
│                         Go Backend (Gin)                                      │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                    Middleware: JWT Auth + Admin Guard                    │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                    REST API Layer (103 routes)                           │  │
│  │  /api/v1/auth/*           /api/v1/dashboard      /api/v1/projections    │  │
│  │  /api/v1/mutual-funds/*   /api/v1/stocks/*       /api/v1/corporate-*    │  │
│  │  /api/v1/fixed-deposits/* /api/v1/provident-*    /api/v1/home-loans/*   │  │
│  │  /api/v1/personal-loans/* /api/v1/nps/*          /api/v1/credit-cards/* │  │
│  │  /api/v1/sips/*           /api/v1/goals/*        /api/v1/net-worth/*    │  │
│  │  /api/v1/cashflow/*       /api/v1/insights       /api/v1/benchmarks     │  │
│  │  /api/v1/tax/*            /api/v1/notifications/* /api/v1/profiles/*    │  │
│  │  /api/v1/export/*         /api/v1/backup/*                              │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐ │
│  │  NAV Fetcher │ │ Bond Schedule│ │ Stock Price  │ │  Monte Carlo Engine  │ │
│  │  (AMFI API)  │ │  Calculator  │ │  (Yahoo Fin) │ │  (2000 simulations)  │ │
│  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────────────┘ │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐ │
│  │  Insights    │ │  Scheduler   │ │  Tax Engine  │ │  Notification Gen    │ │
│  │  Engine      │ │  (Daily 8PM) │ │  (80C/24b)   │ │  (7-day lookahead)   │ │
│  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────────────┘ │
└───────────────────────────┬───────────────────────────────────────────────────┘
                            │
┌───────────────────────────▼───────────────────────────────────────────────────┐
│                      MongoDB (localhost:27017)                                │
│  Database: investment_tracker                                                 │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐     │
│  │    users     │ │ mutual_funds │ │corporate_    │ │ fixed_deposits   │     │
│  │              │ │              │ │bonds         │ │                  │     │
│  ├──────────────┤ ├──────────────┤ ├──────────────┤ ├──────────────────┤     │
│  │provident_    │ │   stocks     │ │ home_loans   │ │ personal_loans   │     │
│  │fund_entries  │ │              │ │              │ │                  │     │
│  ├──────────────┤ ├──────────────┤ ├──────────────┤ ├──────────────────┤     │
│  │nps_accounts  │ │credit_cards  │ │    sips      │ │     goals        │     │
│  ├──────────────┤ ├──────────────┤ ├──────────────┤ ├──────────────────┤     │
│  │net_worth_    │ │notifications │ │  profiles    │ │                  │     │
│  │snapshots     │ │              │ │              │ │                  │     │
│  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────────┘     │
└───────────────────────────────────────────────────────────────────────────────┘
```

---

## API Endpoints

See [README.md](README.md#api-reference) for the complete 103-route API reference.

---

## MongoDB Schemas

### 1. Collection: `users`

```json
{
  "_id": "ObjectId",
  "email": "user@example.com",
  "password_hash": "$2a$10...",
  "name": "John Doe",
  "role": "user",
  "reset_token": "",
  "reset_token_expiry": null,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

### 2. Collection: `mutual_funds`

Tracks each mutual fund holding. Each transaction (SIP purchase, lumpsum, redemption) is stored inside.

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "fund_name": "Axis Long Term Equity Fund",
  "amc": "Axis Mutual Fund",
  "fund_type": "ELSS",
  "scheme_code": "112323",
  "folio_number": "1234567890",
  "is_elss": true,
  "lock_in_end_date": "2029-04-15",
  "transactions": [
    {
      "transaction_id": "UUID",
      "date": "2026-04-15",
      "type": "purchase",
      "amount": 50000.00,
      "nav_at_purchase": 45.23,
      "units": 1105.46,
      "lock_in_end": "2029-04-15"
    }
  ],
  "total_units": 1105.46,
  "total_invested": 50000.00,
  "current_nav": 48.50,
  "current_value": 53614.81,
  "nav_last_updated": "2026-04-19T10:00:00Z",
  "gain_loss": 3614.81,
  "gain_loss_percent": 7.23,
  "notes": "",
  "created_at": "2026-04-15T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

### 3. Collection: `corporate_bonds`

Complex schema to handle different payout frequencies and maturity types (bullet vs staggered).

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "bond_name": "Muthoot Finance NCD 2027",
  "issuer": "Muthoot Finance Ltd",
  "purchase_date": "2025-06-01",
  "investment_amount": 100000.00,
  "coupon_rate": 9.50,
  "interest_payout_frequency": "quarterly",
  "maturity_date": "2027-06-01",
  "maturity_type": "staggered",
  "principal_repayments": [
    { "repayment_id": "UUID", "scheduled_date": "2026-10-01", "amount": 33333.33, "status": "pending", "received_date": null }
  ],
  "interest_payouts": [
    { "payout_id": "UUID", "scheduled_date": "2025-09-01", "principal_at_time": 100000.00, "amount": 2375.00, "status": "received", "received_date": "2025-09-03" }
  ],
  "remaining_principal": 100000.00,
  "total_interest_earned": 4750.00,
  "total_principal_returned": 0.00,
  "status": "active",
  "notes": "",
  "created_at": "2025-06-01T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

### 4. Collection: `fixed_deposits`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "bank_name": "SBI",
  "fd_number": "FD-2024-001",
  "principal_amount": 200000.00,
  "interest_rate": 7.10,
  "start_date": "2025-01-15",
  "maturity_date": "2026-01-15",
  "tenure_months": 12,
  "interest_type": "cumulative",
  "payout_frequency": null,
  "maturity_amount": 214200.00,
  "interest_earned": 14200.00,
  "is_auto_renewed": false,
  "status": "active",
  "notes": "",
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

### 5. Collection: `provident_fund_entries`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "account_type": "EPF",
  "account_number": "MH/PUN/12345/67890",
  "employer_name": "TCS",
  "interest_rate": 8.25,
  "financial_year_entries": [
    {
      "financial_year": "2025-2026",
      "monthly_contributions": [
        { "month": "2025-04", "employee_contribution": 1800.00, "employer_contribution": 1800.00, "total": 3600.00 }
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

### 6. Collection: `stocks`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "stock_name": "Reliance Industries Ltd",
  "symbol": "RELIANCE",
  "exchange": "NSE",
  "transactions": [
    { "transaction_id": "UUID", "date": "2026-01-10", "type": "buy", "quantity": 10, "price_per_share": 2450.00, "amount": 24500.00 }
  ],
  "dividends": [
    { "dividend_id": "UUID", "date": "2026-03-15", "amount_per_share": 8.00, "total_amount": 80.00 }
  ],
  "total_quantity": 10,
  "total_invested": 24500.00,
  "avg_buy_price": 2450.00,
  "current_price": 2580.00,
  "current_value": 25800.00,
  "day_change": 30.00,
  "day_change_percent": 1.18,
  "gain_loss": 1300.00,
  "gain_loss_percent": 5.31,
  "price_last_updated": "2026-04-22T14:30:00Z",
  "notes": "",
  "created_at": "2026-01-10T10:00:00Z",
  "updated_at": "2026-04-22T14:30:00Z"
}
```

### 7. Collection: `home_loans`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "bank_name": "SBI",
  "loan_account_number": "HL-001",
  "property_address": "Flat 301, Pune",
  "loan_purpose": "Purchase",
  "sanctioned_amount": 5000000.00,
  "disbursed_amount": 5000000.00,
  "interest_rate": 8.50,
  "rate_type": "floating",
  "tenure_months": 240,
  "emi_amount": 43391.00,
  "emi_start_date": "2025-01-01",
  "disbursement_date": "2024-12-15",
  "co_borrower": "Spouse Name",
  "is_under_construction": false,
  "full_emi_started": true,
  "disbursements": [],
  "pre_emis_paid": [],
  "rate_change_history": [
    { "effective_date": "2025-07-01", "old_rate": 8.75, "new_rate": 8.50, "new_emi": 43391.00 }
  ],
  "emis_paid": [
    { "emi_id": "UUID", "month": "2025-01", "due_date": "2025-01-05", "emi_amount": 43391.00, "principal_portion": 7808.00, "interest_portion": 35583.00, "outstanding_after": 4992192.00, "paid": true, "paid_date": "2025-01-05" }
  ],
  "prepayments": [
    { "prepayment_id": "UUID", "date": "2025-06-01", "amount": 200000.00, "type": "part_payment", "notes": "" }
  ],
  "outstanding_principal": 4750000.00,
  "total_principal_paid": 250000.00,
  "total_interest_paid": 210000.00,
  "total_amount_paid": 460000.00,
  "total_prepayments": 200000.00,
  "remaining_tenure_months": 228,
  "loan_end_date": "2044-12-01",
  "status": "active",
  "notes": "",
  "created_at": "2024-12-15T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

### 8. Collection: `personal_loans`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "lender_name": "HDFC Bank",
  "loan_account_number": "PL-001",
  "loan_purpose": "Home Renovation",
  "principal_amount": 500000.00,
  "disbursed_amount": 500000.00,
  "interest_rate": 12.50,
  "rate_type": "fixed",
  "tenure_months": 36,
  "emi_amount": 16733.00,
  "emi_start_date": "2025-02-01",
  "disbursement_date": "2025-01-15",
  "processing_fee": 5000.00,
  "foreclosure_charges": 4.00,
  "rate_change_history": [],
  "emis_paid": [],
  "prepayments": [],
  "outstanding_principal": 500000.00,
  "total_principal_paid": 0.00,
  "total_interest_paid": 0.00,
  "total_amount_paid": 0.00,
  "total_prepayments": 0.00,
  "remaining_tenure_months": 36,
  "loan_end_date": "2028-01-01",
  "status": "active",
  "notes": "",
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

### 9. Collection: `nps_accounts`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "account_holder_name": "John Doe",
  "pran": "110012345678",
  "account_type": "tier_1",
  "fund_manager": "SBI Pension Fund",
  "date_of_joining": "2020-01-15",
  "equity_pct": 50.0,
  "corporate_bond_pct": 30.0,
  "govt_sec_pct": 20.0,
  "alternate_pct": 0.0,
  "contributions": [
    { "contribution_id": "UUID", "date": "2026-04-15", "amount": 50000.00, "type": "self", "fy": "2026-2027" }
  ],
  "total_self_contribution": 300000.00,
  "total_employer_contribution": 100000.00,
  "total_contribution": 400000.00,
  "current_value": 520000.00,
  "status": "active",
  "notes": "",
  "created_at": "2020-01-15T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

### 10. Collection: `credit_cards`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "card_name": "HDFC Regalia",
  "bank_name": "HDFC Bank",
  "card_network": "Visa",
  "last_four_digits": "4321",
  "card_holder_name": "John Doe",
  "credit_limit": 500000.00,
  "billing_date": 15,
  "due_date_offset": 20,
  "annual_fee": 2500.00,
  "joining_date": "2023-06-01",
  "current_outstanding": 45000.00,
  "available_credit": 455000.00,
  "reward_points": 12500.00,
  "reward_point_value": 0.30,
  "card_emis": [
    { "emi_id": "UUID", "description": "Laptop", "merchant_name": "Amazon", "original_amount": 80000.00, "emi_amount": 6800.00, "tenure_months": 12, "remaining_months": 8, "interest_rate": 15.0, "processing_fee": 500.00, "start_date": "2025-10-01", "total_paid": 27200.00, "status": "active" }
  ],
  "statements": [
    { "statement_id": "UUID", "month": "2026-04", "statement_date": "2026-04-15", "due_date": "2026-05-05", "total_amount": 45000.00, "minimum_due": 4500.00, "amount_paid": 45000.00, "paid_date": "2026-05-01", "transactions": [], "is_paid": true, "paid_full": true }
  ],
  "credit_scores": [
    { "score_id": "UUID", "date": "2026-03-01", "score": 785, "bureau": "CIBIL", "notes": "" }
  ],
  "status": "active",
  "notes": "",
  "created_at": "2023-06-01T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

### 11. Collection: `sips`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "fund_name": "Mirae Asset Large Cap Fund",
  "fund_id": "mutual-fund-object-id",
  "amc_code": "M",
  "scheme_code": 119598,
  "amount": 5000.00,
  "frequency": "monthly",
  "sip_date": 5,
  "start_date": "2025-01-05",
  "end_date": null,
  "total_installments": 12,
  "completed_installments": 4,
  "missed_installments": 0,
  "total_invested": 20000.00,
  "installments": [
    { "installment_id": "UUID", "date": "2025-01-05", "amount": 5000.00, "nav": 45.23, "units": 110.55, "status": "completed" }
  ],
  "status": "active",
  "notes": "",
  "created_at": "2025-01-01T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

### 12. Collection: `goals`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "name": "Down Payment for Second Home",
  "category": "real_estate",
  "icon": "🏠",
  "priority": "high",
  "target_amount": 2000000.00,
  "target_date": "2028-12-31",
  "assumed_return_rate": 12.0,
  "linked_investments": [
    { "investment_type": "mutual_fund", "investment_id": "mf-object-id", "investment_name": "Axis Bluechip Fund", "allocated_pct": 50.0 },
    { "investment_type": "stock", "investment_id": "stock-object-id", "investment_name": "RELIANCE", "allocated_pct": 30.0 }
  ],
  "status": "active",
  "notes": "",
  "created_at": "2026-01-01T10:00:00Z",
  "updated_at": "2026-04-19T10:00:00Z"
}
```

Computed fields (not stored): `current_value`, `progress_pct`, `monthly_needed`, `months_remaining`, `on_track`, `projected_date`, `shortfall`, `projection_points`.

### 13. Collection: `net_worth_snapshots`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "date": "2026-04-19T20:00:00Z",
  "month": "2026-04",
  "mutual_funds": 500000.00,
  "stocks": 300000.00,
  "fixed_deposits": 200000.00,
  "provident_fund": 400000.00,
  "nps": 150000.00,
  "corporate_bonds": 100000.00,
  "other_assets": 0.00,
  "total_assets": 1650000.00,
  "home_loans": 4750000.00,
  "personal_loans": 450000.00,
  "credit_card_outstanding": 45000.00,
  "other_liabilities": 0.00,
  "total_liabilities": 5245000.00,
  "net_worth": -3595000.00,
  "created_at": "2026-04-19T20:00:00Z"
}
```

### 14. Collection: `notifications`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "type": "emi_due",
  "title": "Home Loan EMI Due",
  "message": "SBI home loan EMI of ₹43,391 is due on May 5, 2026",
  "date": "2026-05-05",
  "reference_type": "home_loan",
  "reference_id": "home-loan-object-id",
  "is_read": false,
  "created_at": "2026-04-28T20:00:00Z"
}
```

### 15. Collection: `profiles`

```json
{
  "_id": "ObjectId",
  "user_id": "user-id-string",
  "name": "Self",
  "relationship": "self",
  "color": "#1a73e8",
  "is_default": true,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

---

## Project Structure

```
investment-tracker/
├── backend/
│   ├── main.go                          # Entry point, 103 routes
│   ├── go.mod
│   ├── config/
│   │   └── config.go                    # MongoDB URI, port, JWT secret
│   ├── database/
│   │   └── mongodb.go                   # Connection setup
│   ├── middleware/
│   │   └── auth.go                      # JWT verification, admin guard
│   ├── models/                          # 18 model files
│   │   ├── user.go                      # User, auth request/response types
│   │   ├── mutual_fund.go
│   │   ├── corporate_bond.go
│   │   ├── fixed_deposit.go
│   │   ├── provident_fund.go
│   │   ├── stock.go
│   │   ├── home_loan.go
│   │   ├── personal_loan.go
│   │   ├── nps.go
│   │   ├── credit_card.go
│   │   ├── sip.go
│   │   ├── goal.go
│   │   ├── net_worth.go
│   │   ├── notification.go
│   │   ├── profile.go
│   │   ├── dashboard.go
│   │   ├── projection.go
│   │   └── tax.go
│   ├── handlers/                        # 22 handler files
│   │   ├── auth_handler.go
│   │   ├── dashboard_handler.go
│   │   ├── mutual_fund_handler.go
│   │   ├── stock_handler.go
│   │   ├── corporate_bond_handler.go
│   │   ├── fixed_deposit_handler.go
│   │   ├── provident_fund_handler.go
│   │   ├── home_loan_handler.go
│   │   ├── personal_loan_handler.go
│   │   ├── nps_handler.go
│   │   ├── credit_card_handler.go
│   │   ├── sip_handler.go
│   │   ├── goal_handler.go
│   │   ├── net_worth_handler.go
│   │   ├── notification_handler.go
│   │   ├── profile_handler.go
│   │   ├── projection_handler.go
│   │   ├── cashflow_handler.go
│   │   ├── insights_handler.go
│   │   ├── benchmark_handler.go
│   │   ├── tax_handler.go
│   │   ├── export_handler.go
│   │   └── backup_handler.go
│   ├── services/                        # 24 service files
│   │   ├── auth_service.go
│   │   ├── dashboard_service.go
│   │   ├── mutual_fund_service.go
│   │   ├── stock_service.go
│   │   ├── corporate_bond_service.go
│   │   ├── fixed_deposit_service.go
│   │   ├── provident_fund_service.go
│   │   ├── home_loan_service.go
│   │   ├── personal_loan_service.go
│   │   ├── nps_service.go
│   │   ├── credit_card_service.go
│   │   ├── goal_service.go
│   │   ├── net_worth_service.go
│   │   ├── notification_service.go
│   │   ├── profile_service.go
│   │   ├── projection_service.go        # Monte Carlo simulation engine
│   │   ├── cashflow_service.go
│   │   ├── insights_service.go
│   │   ├── benchmark_service.go
│   │   ├── export_service.go
│   │   ├── backup_service.go
│   │   ├── nav_fetcher.go               # AMFI NAV auto-fetch
│   │   ├── stock_price_fetcher.go       # Yahoo Finance live prices
│   │   ├── scheduler.go                 # Daily 20:00 IST background job
│   │   └── helpers.go                   # Shared utilities
│   └── repository/                      # 15 repository files
│       ├── user_repo.go
│       ├── mutual_fund_repo.go
│       ├── corporate_bond_repo.go
│       ├── fixed_deposit_repo.go
│       ├── provident_fund_repo.go
│       ├── stock_repo.go
│       ├── home_loan_repo.go
│       ├── personal_loan_repo.go
│       ├── nps_repo.go
│       ├── credit_card_repo.go
│       ├── sip_repo.go
│       ├── goal_repo.go
│       ├── net_worth_repo.go
│       ├── notification_repo.go
│       └── profile_repo.go
│
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   └── src/
│       ├── App.tsx
│       ├── main.tsx
│       ├── api/                         # API client functions
│       │   ├── client.ts                # Axios instance with JWT interceptor
│       │   ├── dashboard.ts
│       │   ├── mutualFunds.ts
│       │   ├── stocks.ts
│       │   ├── corporateBonds.ts
│       │   ├── fixedDeposits.ts
│       │   ├── providentFund.ts
│       │   ├── homeLoans.ts
│       │   ├── personalLoans.ts
│       │   ├── nps.ts
│       │   ├── creditCards.ts
│       │   ├── sips.ts
│       │   ├── goals.ts
│       │   ├── netWorth.ts
│       │   ├── projections.ts
│       │   ├── cashflow.ts
│       │   ├── insights.ts
│       │   ├── benchmarks.ts
│       │   ├── tax.ts
│       │   ├── notifications.ts
│       │   ├── profiles.ts
│       │   ├── backup.ts
│       │   └── auth.ts
│       ├── components/
│       │   └── Sidebar.tsx
│       ├── context/
│       │   └── AuthContext.tsx           # JWT token + user state
│       ├── pages/                       # 21 pages
│       │   ├── LoginPage.tsx
│       │   ├── SignupPage.tsx
│       │   ├── ForgotPasswordPage.tsx
│       │   ├── ResetPasswordPage.tsx
│       │   ├── DashboardPage.tsx
│       │   ├── MutualFundsPage.tsx
│       │   ├── StocksPage.tsx
│       │   ├── CorporateBondsPage.tsx
│       │   ├── FixedDepositsPage.tsx
│       │   ├── ProvidentFundPage.tsx
│       │   ├── HomeLoansPage.tsx
│       │   ├── PersonalLoansPage.tsx
│       │   ├── NPSPage.tsx
│       │   ├── CreditCardsPage.tsx
│       │   ├── GoalsPage.tsx
│       │   ├── NetWorthPage.tsx
│       │   ├── ProjectionsPage.tsx
│       │   ├── CashflowPage.tsx
│       │   ├── InsightsPage.tsx
│       │   ├── TaxCenterPage.tsx
│       │   └── SettingsPage.tsx
│       ├── types/
│       │   └── index.ts                 # TypeScript interfaces (mirrors Go models)
│       └── utils/
│           ├── format.ts                # Currency & date formatting
│           ├── casParser.ts             # CAS PDF parser
│           └── pfParser.ts              # PF passbook PDF parser
│
├── package.json                         # Root scripts (start both servers)
├── FEATURES.txt                         # Complete feature inventory
├── ARCHITECTURE.md                      # This file
└── README.md
```

---

## Key Business Logic

### 1. Authentication & Authorization
- Passwords hashed with bcrypt (cost 10)
- JWT tokens with user ID and role claims
- Middleware extracts user ID from token for all authenticated routes
- Admin role can list users and impersonate via `X-View-As-User` header (read-only)
- Forgot password generates a reset token (stored hashed) with expiry

### 2. Corporate Bond Schedule Generator
When a bond is added, the backend auto-generates:
- **Interest payout schedule**: Based on `interest_payout_frequency`, from `purchase_date` to `maturity_date`
- **Principal repayment schedule**: User enters this manually (staggered dates + amounts) or system sets single bullet repayment at maturity

When a principal repayment is marked as received:
1. `remaining_principal` decreases
2. All **future** interest payouts are recalculated using the new `remaining_principal`
3. `principal_at_time` on future payouts is updated

### 3. Mutual Fund NAV Auto-Fetch
- Uses free AMFI API: `https://api.mfapi.in/mf/{scheme_code}`
- On demand (manual refresh) or via daily scheduler at 20:00 IST
- Updates `current_nav`, `current_value`, `gain_loss`, `gain_loss_percent`
- At data entry time: user manually enters `nav_at_purchase` for existing holdings

### 4. Stock Live Price Fetch
- Uses Yahoo Finance chart API: `https://query1.finance.yahoo.com/v8/finance/chart/{SYMBOL}.NS`
- Symbols suffixed with `.NS` (NSE) or `.BO` (BSE)
- Returns current price, previous close, day change, and day change %
- Frontend auto-refreshes every 60 seconds when market is open
- Market hours: Mon–Fri 9:15 AM – 3:30 PM IST
- `IsMarketOpen()` helper checks current IST time for market status

### 5. Stock Average Cost Calculation
- Uses running average cost method
- Buy: adds to total quantity and total invested
- Sell: deducts quantity and proportional cost at current average price
- `avg_buy_price = total_invested / total_quantity`

### 6. Monte Carlo Projection Engine
- Runs 2,000 simulated growth paths per investment using Geometric Brownian Motion
- Monthly step: `value *= (1 + monthlyDrift + randomShock * monthlyVolatility)`
- Extracts P10 (pessimistic), P50 (median), P90 (optimistic) percentiles at each month
- **Volatility estimation**: Computed from historical transaction NAV/price data where available, otherwise category-specific defaults (18% equity MF, 22% stocks, 3% debt, 1% FD/PF, 10% hybrid)
- **Rate estimation**: Historical CAGR from transaction data, with fallbacks by asset type
- Replaces simple bull/bear multipliers with statistically grounded confidence intervals
- FDs and PF produce tight bands (near-deterministic); stocks and equity MFs produce wide bands
- Supports 1–30 year projection horizons

### 7. Home Loan Under-Construction Support
- Tracks tranche disbursements with pre-EMI interest calculation
- Pre-EMI = daily interest on disbursed amount for remaining days in month
- On `mark-complete`: switches from pre-EMI to full EMI mode, recalculates schedule

### 8. Insights Engine
- Scans entire portfolio to generate actionable insights
- Categories: risk, performance, tax, diversification, loan, goal, opportunity
- Severity levels: critical, warning, info, positive
- Examples: concentration risk, unused 80C limit, high-interest loans, underperforming funds

### 9. Tax Computation
- **Section 80C** (₹1.5L limit): EPF contributions + ELSS investments + home loan principal + PPF + NPS (80CCD1)
- **Section 24b** (₹2L limit): Home loan interest deduction
- **Section 80CCD(1B)** (₹50K additional): NPS self-contribution
- **Capital gains**: Tracks LTCG/STCG from MF redemptions and stock sells

### 10. Notification Generator
- Scans all modules for events within next 7 days
- Event types: EMI due dates, FD maturity, bond interest/principal payouts, SIP installment dates, credit card due dates, goal deadlines
- Deduplicates against existing notifications

### 11. Daily Scheduler
- Runs at 20:00 IST daily (after Indian market close)
- Refreshes NAV for all mutual funds
- Refreshes stock prices for all holdings
- Takes a net worth snapshot for historical tracking

### 12. ELSS Lock-in Tracking
- Each ELSS transaction has its own `lock_in_end` (purchase_date + 3 years)
- Dashboard shows which units are locked vs unlocked
- Filter view for ELSS-only investments

### 13. FD Maturity Calculation
- Cumulative: compound interest calculation for maturity amount
- Non-cumulative: simple interest per payout period

### 14. PF Interest Calculation
- Monthly running balance method (as per EPFO rules)
- Interest calculated on monthly closing balances
- Credited annually

---

## Dashboard Summary

The dashboard aggregates across all investment and liability types:

| Metric                     | Source                                                                         |
|----------------------------|--------------------------------------------------------------------------------|
| Total Invested             | Sum across MF + Stocks + Bonds + FD + PF + NPS                                |
| Current Value              | MF (units×NAV) + Stocks (qty×price) + FD + Bonds + PF + NPS current values    |
| Total Gains                | Current Value - Total Invested                                                 |
| Overall Return %           | (Gains / Invested) × 100                                                       |
| Portfolio XIRR             | Time-weighted return from all transaction cashflows                             |
| Risk Score                 | Composite of equity/debt ratio, diversification, concentration, loan-to-asset  |
| Upcoming Payouts           | Next 30 days: bond interest, FD maturity, bond principal                       |
| Loan Summary               | Total outstanding, total EMI, total prepayments (home + personal)              |
| Credit Card Summary        | Total outstanding, total limit, avg utilization                                |
| ELSS Tax Saving (80C)      | Sum of ELSS investments in current FY (max ₹1.5L)                             |
| Asset Allocation Pie       | % in MF, Stocks, Bonds, FD, PF, NPS                                           |
| Rebalancing Suggestions    | User provides target allocation → system suggests buy/sell actions             |

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
