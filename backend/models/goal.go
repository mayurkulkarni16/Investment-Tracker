package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LinkedInvestment struct {
	InvestmentType string  `json:"investment_type" bson:"investment_type"`
	InvestmentID   string  `json:"investment_id" bson:"investment_id"`
	InvestmentName string  `json:"investment_name" bson:"investment_name"`
	AllocatedPct   float64 `json:"allocated_pct" bson:"allocated_pct"`
}

type GoalProjection struct {
	Month int     `json:"month"`
	Value float64 `json:"value"`
}

type Goal struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name     string             `json:"name" bson:"name"`
	Category string             `json:"category" bson:"category"` // retirement, education, house, car, wedding, emergency, travel, other
	Icon     string             `json:"icon" bson:"icon"`         // emoji
	Priority string             `json:"priority" bson:"priority"` // high, medium, low

	TargetAmount      float64   `json:"target_amount" bson:"target_amount"`
	TargetDate        time.Time `json:"target_date" bson:"target_date"`
	AssumedReturnRate float64   `json:"assumed_return_rate" bson:"assumed_return_rate"` // annual %

	LinkedInvestments []LinkedInvestment `json:"linked_investments" bson:"linked_investments"`

	// Computed (not stored)
	CurrentValue     float64          `json:"current_value" bson:"-"`
	ProgressPct      float64          `json:"progress_pct" bson:"-"`
	MonthlyNeeded    float64          `json:"monthly_needed" bson:"-"`
	MonthsRemaining  int              `json:"months_remaining" bson:"-"`
	OnTrack          bool             `json:"on_track" bson:"-"`
	ProjectedDate    *time.Time       `json:"projected_date,omitempty" bson:"-"`
	Shortfall        float64          `json:"shortfall" bson:"-"`
	ProjectionPoints []GoalProjection `json:"projection_points,omitempty" bson:"-"`

	Status    string    `json:"status" bson:"status"` // active, achieved, abandoned
	Notes     string    `json:"notes" bson:"notes"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type CreateGoalRequest struct {
	Name              string  `json:"name" binding:"required"`
	Category          string  `json:"category" binding:"required"`
	Icon              string  `json:"icon"`
	Priority          string  `json:"priority"`
	TargetAmount      float64 `json:"target_amount" binding:"required"`
	TargetDate        string  `json:"target_date" binding:"required"`
	AssumedReturnRate float64 `json:"assumed_return_rate"`
	Notes             string  `json:"notes"`
}

type UpdateGoalRequest struct {
	Name              string  `json:"name"`
	Category          string  `json:"category"`
	Icon              string  `json:"icon"`
	Priority          string  `json:"priority"`
	TargetAmount      float64 `json:"target_amount"`
	TargetDate        string  `json:"target_date"`
	AssumedReturnRate float64 `json:"assumed_return_rate"`
	Status            string  `json:"status"`
	Notes             string  `json:"notes"`
}

type LinkInvestmentRequest struct {
	InvestmentType string  `json:"investment_type" binding:"required"`
	InvestmentID   string  `json:"investment_id" binding:"required"`
	InvestmentName string  `json:"investment_name" binding:"required"`
	AllocatedPct   float64 `json:"allocated_pct" binding:"required"`
}

type BatchLinkInvestmentRequest struct {
	Investments []LinkInvestmentRequest `json:"investments" binding:"required"`
}
