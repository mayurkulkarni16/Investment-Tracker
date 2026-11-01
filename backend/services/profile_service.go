package services

import (
	"context"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson"
)

var relationships = map[string]bool{"self": true, "spouse": true, "parent": true, "child": true, "sibling": true, "other": true}

type ProfileService struct {
	repo       *repository.ProfileRepo
	assignRepo *repository.ProfileAssignmentRepo
	incomeRepo *repository.MemberIncomeRepo
}

func NewProfileService(repo *repository.ProfileRepo) *ProfileService {
	return &ProfileService{repo: repo}
}

// SetCleanupRepos lets Delete remove the member's ownership tags and income records.
func (s *ProfileService) SetCleanupRepos(assignRepo *repository.ProfileAssignmentRepo, incomeRepo *repository.MemberIncomeRepo) {
	s.assignRepo, s.incomeRepo = assignRepo, incomeRepo
}

func validateTaxFields(occupation, residency, disability, regime string) error {
	if occupation != "" {
		if _, ok := models.Occupations[occupation]; !ok {
			return invalidf("unknown occupation %q", occupation)
		}
	}
	switch residency {
	case "", "resident", "nri":
	default:
		return invalidf("residential_status must be resident or nri")
	}
	switch disability {
	case "", "disabled", "severe":
	default:
		return invalidf("disability must be disabled or severe")
	}
	switch regime {
	case "", "old", "new":
	default:
		return invalidf("preferred_regime must be old or new")
	}
	return nil
}

func parseDOB(v string) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, invalidf("date_of_birth must be YYYY-MM-DD")
	}
	if t.After(time.Now()) || t.Year() < 1900 {
		return nil, invalidf("date_of_birth is out of range")
	}
	return &t, nil
}

func withAge(p *models.Profile) *models.Profile {
	p.Age = p.AgeOn(time.Now())
	return p
}

func (s *ProfileService) Create(ctx context.Context, userID string, req models.CreateProfileRequest) (*models.Profile, error) {
	if !relationships[req.Relationship] {
		return nil, invalidf("relationship must be self, spouse, parent, child, sibling or other")
	}
	if err := validateTaxFields(req.Occupation, req.ResidentialStatus, req.Disability, req.PreferredRegime); err != nil {
		return nil, err
	}
	dob, err := parseDOB(req.DateOfBirth)
	if err != nil {
		return nil, err
	}
	color := req.Color
	if color == "" {
		color = "#4CAF50"
	}

	profile := &models.Profile{
		UserID:            userID,
		Name:              strings.TrimSpace(req.Name),
		Relationship:      req.Relationship,
		Color:             color,
		IsDefault:         req.IsDefault,
		Occupation:        req.Occupation,
		DateOfBirth:       dob,
		ResidentialStatus: req.ResidentialStatus,
		Disability:        req.Disability,
		SpouseProfileID:   req.SpouseProfileID,
		PreferredRegime:   req.PreferredRegime,
		MetroCity:         req.MetroCity,
	}

	if err := s.repo.Create(ctx, profile); err != nil {
		return nil, err
	}
	return withAge(profile), nil
}

func (s *ProfileService) GetAll(ctx context.Context, userID string) ([]models.Profile, error) {
	profiles, err := s.repo.GetAll(ctx, userID)
	for i := range profiles {
		withAge(&profiles[i])
	}
	return profiles, err
}

func (s *ProfileService) Update(ctx context.Context, id string, req models.UpdateProfileRequest) (*models.Profile, error) {
	objID, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	profile, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		profile.Name = strings.TrimSpace(req.Name)
	}
	if req.Relationship != "" {
		if !relationships[req.Relationship] {
			return nil, invalidf("relationship must be self, spouse, parent, child, sibling or other")
		}
		profile.Relationship = req.Relationship
	}
	if req.Color != "" {
		profile.Color = req.Color
	}
	str := func(p *string, cur string) string {
		if p == nil {
			return cur
		}
		return strings.TrimSpace(*p)
	}
	occupation := str(req.Occupation, profile.Occupation)
	residency := str(req.ResidentialStatus, profile.ResidentialStatus)
	disability := str(req.Disability, profile.Disability)
	regime := str(req.PreferredRegime, profile.PreferredRegime)
	if err := validateTaxFields(occupation, residency, disability, regime); err != nil {
		return nil, err
	}
	profile.Occupation, profile.ResidentialStatus, profile.Disability, profile.PreferredRegime = occupation, residency, disability, regime
	if req.DateOfBirth != nil {
		dob, err := parseDOB(*req.DateOfBirth)
		if err != nil {
			return nil, err
		}
		profile.DateOfBirth = dob
	}
	if req.SpouseProfileID != nil {
		if *req.SpouseProfileID == id {
			return nil, invalidf("a member cannot be their own spouse")
		}
		profile.SpouseProfileID = *req.SpouseProfileID
	}
	if req.MetroCity != nil {
		profile.MetroCity = *req.MetroCity
	}

	if err := s.repo.Update(ctx, profile); err != nil {
		return nil, err
	}
	return withAge(profile), nil
}

// Delete removes the profile and every reference to it: income records,
// ownership tags (joint shares are rescaled to 100%), "funded by" links and
// spouse links.
func (s *ProfileService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	profile, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, objID); err != nil {
		return err
	}
	userID := profile.UserID

	if s.incomeRepo != nil {
		_ = s.incomeRepo.DeleteMany(ctx, bson.M{"user_id": userID, "profile_id": id})
	}
	if s.assignRepo != nil {
		assignments, _ := s.assignRepo.GetAll(ctx, userID)
		for _, a := range assignments {
			shares := a.EffectiveShares()
			kept := make([]models.OwnershipShare, 0, len(shares))
			for _, sh := range shares {
				if sh.ProfileID != id {
					kept = append(kept, sh)
				}
			}
			fundedBy := a.FundedBy
			if fundedBy == id {
				fundedBy = ""
			}
			if len(kept) == len(shares) && fundedBy == a.FundedBy {
				continue
			}
			if len(kept) == 0 {
				_ = s.assignRepo.Delete(ctx, a.ID)
				continue
			}
			kept = normalizeShares(kept)
			_ = s.assignRepo.Set(ctx, a.ID, bson.M{"shares": kept, "profile_id": primaryOwner(kept), "funded_by": fundedBy})
		}
	}
	others, _ := s.repo.GetAll(ctx, userID)
	for i := range others {
		if others[i].SpouseProfileID == id {
			others[i].SpouseProfileID = ""
			_ = s.repo.Update(ctx, &others[i])
		}
	}
	return nil
}
