package services

import (
	"context"
	"math"
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
	byAsset := map[string]*models.ProfileAssignment{}
	for i := range assignments {
		a := &assignments[i]
		byAsset[a.AssetType+":"+a.AssetID] = a
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
		it.Shares = []models.OwnershipShare{}
		var shares []models.OwnershipShare
		if a := byAsset[it.AssetType+":"+it.AssetID]; a != nil {
			for _, sh := range a.EffectiveShares() {
				if _, ok := members[sh.ProfileID]; ok { // skip deleted profiles
					shares = append(shares, sh)
				}
			}
			if _, ok := members[a.FundedBy]; ok {
				it.FundedBy = a.FundedBy
			}
		}
		if len(shares) == 0 {
			shares = []models.OwnershipShare{{ProfileID: "", Pct: 100}}
		} else {
			shares = normalizeShares(shares)
			it.Shares = shares
			it.ProfileID = primaryOwner(shares)
		}
		for _, sh := range shares {
			m := members[sh.ProfileID]
			if m == nil {
				m = newMember("", "Unassigned", "", "#94a3b8")
			}
			v := it.Value * sh.Pct / 100
			m.ItemCount++
			if it.IsLiability {
				m.Liabilities[it.Category] = round2(m.Liabilities[it.Category] + v)
				m.TotalLiabilities += v
			} else {
				m.Assets[it.Category] = round2(m.Assets[it.Category] + v)
				m.TotalAssets += v
			}
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
	shares := req.Shares
	if len(shares) == 0 && req.ProfileID != "" {
		shares = []models.OwnershipShare{{ProfileID: req.ProfileID, Pct: 100}}
	}
	if len(shares) == 0 {
		return s.assignRepo.DeleteMany(ctx, filter)
	}

	seen := map[string]bool{}
	total := 0.0
	for _, sh := range shares {
		if sh.Pct <= 0 || sh.Pct > 100 {
			return invalidf("each share must be between 0 and 100%%")
		}
		if seen[sh.ProfileID] {
			return invalidf("a member can appear only once in the ownership split")
		}
		seen[sh.ProfileID] = true
		total += sh.Pct
		if err := s.checkProfile(ctx, sh.ProfileID); err != nil {
			return err
		}
	}
	if math.Abs(total-100) > 0.01 {
		return invalidf("ownership shares must add up to 100%% (got %.2f%%)", total)
	}
	if req.FundedBy != "" {
		if err := s.checkProfile(ctx, req.FundedBy); err != nil {
			return err
		}
		if len(shares) == 1 && shares[0].ProfileID == req.FundedBy {
			req.FundedBy = "" // funded by the sole owner = own money
		}
	}
	shares = normalizeShares(shares)
	return s.assignRepo.Upsert(ctx, filter, bson.M{"user_id": userID, "asset_type": req.AssetType, "asset_id": req.AssetID,
		"profile_id": primaryOwner(shares), "shares": shares, "funded_by": req.FundedBy})
}

func (s *FamilyService) checkProfile(ctx context.Context, id string) error {
	pid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return invalidf("invalid profile_id")
	}
	if _, err := s.profileRepo.GetByID(ctx, pid); err != nil {
		return notFoundf("profile not found")
	}
	return nil
}

// normalizeShares rescales shares to total 100% and orders them largest first.
func normalizeShares(shares []models.OwnershipShare) []models.OwnershipShare {
	total := 0.0
	for _, sh := range shares {
		total += sh.Pct
	}
	out := make([]models.OwnershipShare, 0, len(shares))
	for _, sh := range shares {
		if total > 0 {
			sh.Pct = math.Round(sh.Pct/total*10000) / 100
		}
		out = append(out, sh)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Pct > out[j].Pct })
	return out
}

func primaryOwner(shares []models.OwnershipShare) string {
	if len(shares) == 0 {
		return ""
	}
	return shares[0].ProfileID
}
