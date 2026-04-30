package services

import (
	"context"

	"investment-tracker/models"
	"investment-tracker/repository"
)

type ProfileService struct {
	repo *repository.ProfileRepo
}

func NewProfileService(repo *repository.ProfileRepo) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) Create(ctx context.Context, userID string, req models.CreateProfileRequest) (*models.Profile, error) {
	color := req.Color
	if color == "" {
		color = "#4CAF50"
	}

	profile := &models.Profile{
		UserID:       userID,
		Name:         req.Name,
		Relationship: req.Relationship,
		Color:        color,
		IsDefault:    req.IsDefault,
	}

	if err := s.repo.Create(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *ProfileService) GetAll(ctx context.Context, userID string) ([]models.Profile, error) {
	return s.repo.GetAll(ctx, userID)
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
		profile.Name = req.Name
	}
	if req.Relationship != "" {
		profile.Relationship = req.Relationship
	}
	if req.Color != "" {
		profile.Color = req.Color
	}

	if err := s.repo.Update(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *ProfileService) Delete(ctx context.Context, id string) error {
	objID, err := parseObjectID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}
