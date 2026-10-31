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
	ProfileID   string  `json:"profile_id"` // "" = unassigned
}

// ProfileAssignment tags an asset or liability to a family member profile.
type ProfileAssignment struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    string             `json:"user_id" bson:"user_id"`
	AssetType string             `json:"asset_type" bson:"asset_type"`
	AssetID   string             `json:"asset_id" bson:"asset_id"`
	ProfileID string             `json:"profile_id" bson:"profile_id"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

type AssignProfileRequest struct {
	AssetType string `json:"asset_type" binding:"required"`
	AssetID   string `json:"asset_id" binding:"required"`
	ProfileID string `json:"profile_id"` // empty to unassign
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
