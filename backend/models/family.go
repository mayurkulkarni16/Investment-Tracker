package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AssetItem is one asset or liability with its current value, used by net
// worth, family (per-member) views and document attachments.
type AssetItem struct {
	AssetType   string  `json:"asset_type"` // mutual_fund, stock, fixed_deposit, provident_fund, nps, corporate_bond, vehicle_rental, bank_account, gold, property, insurance, home_loan, personal_loan, credit_card
	AssetID     string  `json:"asset_id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"` // net worth bucket, e.g. "Mutual Funds"
	Value       float64 `json:"value"`
	IsLiability bool    `json:"is_liability"`
	ProfileID   string  `json:"profile_id"` // "" = unassigned; primary owner when shared
	// Shares splits ownership between members (sums to 100). Empty when unassigned.
	Shares   []OwnershipShare `json:"shares"`
	FundedBy string           `json:"funded_by"` // member whose money paid for it ("" = the owners themselves)
}

type OwnershipShare struct {
	ProfileID string  `json:"profile_id" bson:"profile_id"`
	Pct       float64 `json:"pct" bson:"pct"`
}

// ProfileAssignment tags an asset or liability to one or more family members.
type ProfileAssignment struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    string             `json:"user_id" bson:"user_id"`
	AssetType string             `json:"asset_type" bson:"asset_type"`
	AssetID   string             `json:"asset_id" bson:"asset_id"`
	ProfileID string             `json:"profile_id" bson:"profile_id"` // primary (largest share) owner
	// Shares is empty for legacy single-owner rows, meaning 100% to ProfileID.
	Shares []OwnershipShare `json:"shares" bson:"shares,omitempty"`
	// FundedBy is the member whose money paid for the asset when it differs
	// from the owner(s). It drives clubbing of income (spouse, minor child)
	// and who claims deductions such as 80C/80D.
	FundedBy  string    `json:"funded_by" bson:"funded_by,omitempty"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// EffectiveShares returns the ownership split, treating legacy rows as 100%.
func (a *ProfileAssignment) EffectiveShares() []OwnershipShare {
	if len(a.Shares) > 0 {
		return a.Shares
	}
	if a.ProfileID == "" {
		return nil
	}
	return []OwnershipShare{{ProfileID: a.ProfileID, Pct: 100}}
}

type AssignProfileRequest struct {
	AssetType string           `json:"asset_type" binding:"required"`
	AssetID   string           `json:"asset_id" binding:"required"`
	ProfileID string           `json:"profile_id"` // single owner; empty (with no shares) to unassign
	Shares    []OwnershipShare `json:"shares"`     // joint ownership; overrides profile_id
	FundedBy  string           `json:"funded_by"`
}

type MemberSummary struct {
	ProfileID        string             `json:"profile_id"` // "" for unassigned
	Name             string             `json:"name"`
	Relationship     string             `json:"relationship"`
	Color            string             `json:"color"`
	TotalAssets      float64            `json:"total_assets"`
	TotalLiabilities float64            `json:"total_liabilities"`
	NetWorth         float64            `json:"net_worth"`
	Assets           map[string]float64 `json:"assets"`
	Liabilities      map[string]float64 `json:"liabilities"`
	ItemCount        int                `json:"item_count"`
}

type FamilySummary struct {
	Members []MemberSummary `json:"members"`
	Items   []AssetItem     `json:"items"`
}
