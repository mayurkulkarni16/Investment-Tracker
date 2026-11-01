package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Occupations recognised for tax purposes.
var Occupations = map[string]string{
	"salaried_private": "Salaried (private)",
	"salaried_govt":    "Salaried (government)",
	"professional":     "Self-employed professional",
	"business":         "Business owner",
	"pensioner":        "Pensioner / retired",
	"homemaker":        "Homemaker",
	"student":          "Student",
	"minor":            "Minor child",
	"unemployed":       "Not working",
	"other":            "Other",
}

type Profile struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       string             `json:"user_id" bson:"user_id"`
	Name         string             `json:"name" bson:"name"`
	Relationship string             `json:"relationship" bson:"relationship"` // self, spouse, parent, child, sibling, other
	Color        string             `json:"color" bson:"color"`               // hex color for UI
	IsDefault    bool               `json:"is_default" bson:"is_default"`

	// Tax profile (all optional)
	Occupation        string     `json:"occupation" bson:"occupation,omitempty"` // see Occupations
	DateOfBirth       *time.Time `json:"date_of_birth,omitempty" bson:"date_of_birth,omitempty"`
	ResidentialStatus string     `json:"residential_status" bson:"residential_status,omitempty"` // resident (default), nri
	Disability        string     `json:"disability" bson:"disability,omitempty"`                 // "", disabled, severe (80U)
	SpouseProfileID   string     `json:"spouse_profile_id" bson:"spouse_profile_id,omitempty"`
	PreferredRegime   string     `json:"preferred_regime" bson:"preferred_regime,omitempty"` // "" (auto), old, new
	MetroCity         bool       `json:"metro_city" bson:"metro_city,omitempty"`             // lives in a metro (HRA 50%)

	Age int `json:"age,omitempty" bson:"-"` // computed on read

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// IsResident reports whether the member is resident (the default).
func (p *Profile) IsResident() bool { return p.ResidentialStatus != "nri" }

// AgeOn returns the age in completed years on date d (0 if DOB is unknown).
func (p *Profile) AgeOn(d time.Time) int {
	if p.DateOfBirth == nil {
		return 0
	}
	b := *p.DateOfBirth
	age := d.Year() - b.Year()
	if d.Month() < b.Month() || (d.Month() == b.Month() && d.Day() < b.Day()) {
		age--
	}
	if age < 0 {
		return 0
	}
	return age
}

type CreateProfileRequest struct {
	Name              string `json:"name" binding:"required"`
	Relationship      string `json:"relationship" binding:"required"`
	Color             string `json:"color"`
	IsDefault         bool   `json:"is_default"`
	Occupation        string `json:"occupation"`
	DateOfBirth       string `json:"date_of_birth"` // YYYY-MM-DD
	ResidentialStatus string `json:"residential_status"`
	Disability        string `json:"disability"`
	SpouseProfileID   string `json:"spouse_profile_id"`
	PreferredRegime   string `json:"preferred_regime"`
	MetroCity         bool   `json:"metro_city"`
}

// UpdateProfileRequest leaves fields that are omitted (nil) unchanged; an
// empty string clears an optional field.
type UpdateProfileRequest struct {
	Name              string  `json:"name"`
	Relationship      string  `json:"relationship"`
	Color             string  `json:"color"`
	Occupation        *string `json:"occupation"`
	DateOfBirth       *string `json:"date_of_birth"`
	ResidentialStatus *string `json:"residential_status"`
	Disability        *string `json:"disability"`
	SpouseProfileID   *string `json:"spouse_profile_id"`
	PreferredRegime   *string `json:"preferred_regime"`
	MetroCity         *bool   `json:"metro_city"`
}
