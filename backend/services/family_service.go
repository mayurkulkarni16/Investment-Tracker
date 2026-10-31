package services

import (
	"context"
	"sort"

	"investment-tracker/models"
	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var assignableTypes = map[string]bool{
	"mutual_fund": true, "stock": true, "fixed_deposit": true, "provident_fund": true, "nps": true,
	"corporate_bond": true, "vehicle_rental": true, "bank_account": true, "gold": true, "property": true,
	"insurance": true, "home_loan": true, "personal_loan": true, "credit_card": true,
}

// FamilyService tags assets to family member profiles and summarises net
// worth per member.
type FamilyService struct {
	netWorth    *NetWorthService
	profileRepo *repository.ProfileRepo
	assignRepo  *repository.ProfileAssignmentRepo
}

func NewFamilyService(netWorth *NetWorthService, profileRepo *repository.ProfileRepo, assignRepo *repository.ProfileAssignmentRepo) *FamilyService {
	return &FamilyService{netWorth: netWorth, profileRepo: profileRepo, assignRepo: assignRepo}
}

func (s *FamilyService) Summary(ctx context.Context, userID string) (*models.FamilySummary, error) {
	items := s.netWorth.ListItems(ctx, userID)
	assignments, err := s.assignRepo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	profiles, err := s.profileRepo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	byAsset := map[string]string{}
	for _, a := range assignments {
		byAsset[a.AssetType+":"+a.AssetID] = a.ProfileID
	}

	members := map[string]*models.MemberSummary{}
	order := []string{}
	newMember := func(id, name, rel, color string) *models.MemberSummary {
		m := &models.MemberSummary{ProfileID: id, Name: name, Relationship: rel, Color: color,
			Assets: map[string]float64{}, Liabilities: map[string]float64{}}
		members[id] = m
		order = append(order, id)
		return m
	}
	for _, p := range profiles {
		newMember(p.ID.Hex(), p.Name, p.Relationship, p.Color)
	}

	for i := range items {
		it := &items[i]
		pid := byAsset[it.AssetType+":"+it.AssetID]
		if _, ok := members[pid]; !ok {
			pid = "" // unassigned or the profile was deleted
		}
		it.ProfileID = pid
		m := members[pid]
		if m == nil {
			m = newMember("", "Unassigned", "", "#94a3b8")
		}
		m.ItemCount++
		if it.IsLiability {
			m.Liabilities[it.Category] = round2(m.Liabilities[it.Category] + it.Value)
			m.TotalLiabilities += it.Value
		} else {
			m.Assets[it.Category] = round2(m.Assets[it.Category] + it.Value)
			m.TotalAssets += it.Value
		}
	}

	out := &models.FamilySummary{Items: items, Members: []models.MemberSummary{}}
	if out.Items == nil {
		out.Items = []models.AssetItem{}
	}
	for _, id := range order {
		m := members[id]
		m.TotalAssets, m.TotalLiabilities = round2(m.TotalAssets), round2(m.TotalLiabilities)
		m.NetWorth = round2(m.TotalAssets - m.TotalLiabilities)
		out.Members = append(out.Members, *m)
	}
	sort.SliceStable(out.Items, func(i, j int) bool {
		if out.Items[i].Category != out.Items[j].Category {
			return out.Items[i].Category < out.Items[j].Category
		}
		return out.Items[i].Value > out.Items[j].Value
	})
	return out, nil
}

func (s *FamilyService) Assign(ctx context.Context, userID string, req models.AssignProfileRequest) error {
	if !assignableTypes[req.AssetType] {
		return invalidf("unsupported asset_type %q", req.AssetType)
	}
	if _, err := primitive.ObjectIDFromHex(req.AssetID); err != nil {
		return invalidf("invalid asset_id")
	}
	filter := bson.M{"user_id": userID, "asset_type": req.AssetType, "asset_id": req.AssetID}
	if req.ProfileID == "" {
		return s.assignRepo.DeleteMany(ctx, filter)
	}
	pid, err := primitive.ObjectIDFromHex(req.ProfileID)
	if err != nil {
		return invalidf("invalid profile_id")
	}
	if _, err := s.profileRepo.GetByID(ctx, pid); err != nil {
		return notFoundf("profile not found")
	}
	return s.assignRepo.Upsert(ctx, filter, bson.M{"user_id": userID, "asset_type": req.AssetType, "asset_id": req.AssetID, "profile_id": req.ProfileID})
}
