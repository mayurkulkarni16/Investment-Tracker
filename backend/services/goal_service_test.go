package services

import (
	"math"
	"testing"
)

func TestPlanGoalRequiredContributionReachesTarget(t *testing.T) {
	base := goalInputs{target: 1000000, rate: 12, months: 120}
	p := planGoal(base)
	if p.onTrack || math.Abs(p.shortfall-1000000) > 1 {
		t.Fatalf("no savings: onTrack=%v shortfall=%.0f, want false / 10L", p.onTrack, p.shortfall)
	}
	if p.monthlyNeeded < 4000 || p.monthlyNeeded > 4600 {
		t.Errorf("monthly needed for 10L in 10y at 12%% = %.0f, want ~4,300", p.monthlyNeeded)
	}

	withSIP := base
	withSIP.monthly = p.monthlyNeeded
	q := planGoal(withSIP)
	if !q.onTrack || q.shortfall > 1 || q.additionalNeeded > 0.01 {
		t.Errorf("investing the required amount: onTrack=%v shortfall=%.2f additional=%.2f", q.onTrack, q.shortfall, q.additionalNeeded)
	}
	if q.reachMonth < 118 || q.reachMonth > 120 {
		t.Errorf("target reached at month %d, want ~120", q.reachMonth)
	}
}

func TestPlanGoalCountsGrowthOfCurrentValue(t *testing.T) {
	// 5L growing at 12% doubles in ~6.1 years, so a 10L goal in 8 years needs nothing more.
	p := planGoal(goalInputs{target: 1000000, rate: 12, current: 500000, months: 96})
	if !p.onTrack || p.monthlyNeeded != 0 || p.shortfall != 0 {
		t.Errorf("onTrack=%v needed=%.0f shortfall=%.0f, want true/0/0", p.onTrack, p.monthlyNeeded, p.shortfall)
	}
	if p.reachMonth < 70 || p.reachMonth > 76 {
		t.Errorf("reach month = %d, want ~73", p.reachMonth)
	}
	if p.progressPct != 50 {
		t.Errorf("progress = %.2f%%, want 50%%", p.progressPct)
	}
}

func TestPlanGoalInflation(t *testing.T) {
	p := planGoal(goalInputs{target: 1000000, inflation: 6, rate: 10, current: 1000000, months: 120})
	if math.Abs(p.inflatedTarget-1790848) > 100 {
		t.Errorf("inflated target = %.0f, want ~17.9L", p.inflatedTarget)
	}
	if p.progressPct > 56 || p.progressPct < 55 {
		t.Errorf("progress vs inflated target = %.2f%%, want ~55.8%%", p.progressPct)
	}
	// 10L at 10% for 10 years = ~25.9L > 17.9L
	if !p.onTrack {
		t.Error("expected on track")
	}
}
