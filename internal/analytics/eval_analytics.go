package analytics

import (
	"math"
	"strings"
	"time"
)

// EvaluatorResult captures metrics for a single evaluation run
type EvaluatorResult struct {
	RunID               string    `json:"run_id"`
	Timestamp           time.Time `json:"timestamp"`
	ScenarioID          string    `json:"scenario_id"`
	Mode                string    `json:"mode"`
	Success             bool      `json:"success"`
	FalseSuccessClaim   bool      `json:"false_success_claim"`
	RepairIterations    int       `json:"repair_iterations"`
	ClarificationTurns  int       `json:"clarification_turns"`
	Deterministic       bool      `json:"deterministic"`
	ContextPreserved    bool      `json:"context_preserved"`
	InputCount          int       `json:"input_count"`
	OutputCount         int       `json:"output_count"`
	ContextTokens       int64     `json:"context_tokens"`
	IrrelevantRatio     float64   `json:"irrelevant_ratio"` // Ratio of filtered items
	ZeroRecall          bool      `json:"zero_recall"`
	HallucinatedFallback bool      `json:"hallucinated_fallback"`
	EvidenceValidated   bool      `json:"evidence_validated"`
	MaxIterViolation    bool      `json:"max_iter_violation"`
	SemanticHit         bool      `json:"semantic_hit"`
	SemanticFalsePos    bool      `json:"semantic_false_pos"`
	AuthorityOverride   bool      `json:"authority_override"`
	TokensUsed          int64     `json:"tokens_used"`
	RetrievedIDs        []string  `json:"retrieved_ids"` // IDs of memories in context

	// Fields for control run
	MemoryQueries        int   `json:"memory_queries"`
	MemorySemanticHits   int   `json:"memory_semantic_hits"`
	MemoryIrrelevantFiltered int `json:"memory_irrelevant_filtered"`
	RalphIterations      int   `json:"ralph_iterations"`
	RalphEvidenceVerified bool  `json:"ralph_evidence_verified"`
	FilesAPIModified     bool  `json:"files_api_modified"`
	FilesInternalModified bool  `json:"files_internal_modified"`
	FilesUnrelatedModified bool `json:"files_unrelated_modified"`
	TokensApproxPrompt   int64 `json:"tokens_approx_prompt"`
	TokensApproxContext  int64 `json:"tokens_approx_context"`
}

// ModeMetrics aggregates metrics for a specific mode
type ModeMetrics struct {
	Mode                   string  `json:"mode"`
	TotalRuns              int     `json:"total_runs"`
	SuccessRate            float64 `json:"success_rate"`
	TrueSuccessRate        float64 `json:"true_success_rate"` // Adjusted for false successes
	FalseSuccessRate       float64 `json:"false_success_rate"`
	TokensPerSuccess       float64 `json:"tokens_per_success"`
	AvgRetriesPerSuccess   float64 `json:"avg_retries_per_success"`
	AvgContextTokens       float64 `json:"avg_context_tokens"`
	AvgIrrelevantRatio     float64 `json:"avg_irrelevant_ratio"`
	SemanticHitRate        float64 `json:"semantic_hit_rate"`
	StdDevTokensPerSuccess float64 `json:"std_dev_tokens_per_success"`
	
	// Raw aggregations for further calculation
	TotalTokens            int64   `json:"total_tokens"`
	SuccessfulRuns         int     `json:"successful_runs"`
	TrueSuccessfulRuns     int     `json:"true_successful_runs"`
	FalseSuccessClaims     int     `json:"false_success_claims"`
	TotalRepairIterations  int     `json:"total_repair_iterations"`
	TotalContextTokens     int64   `json:"total_context_tokens"`
	TotalIrrelevantRatio   float64 `json:"total_irrelevant_ratio"`
	TotalSemanticHits      int     `json:"total_semantic_hits"`
	TokensPerSuccessValues []int64 `json:"-"` // Used for StdDev
}

// PairwiseDelta represents the comparison between two modes
type PairwiseDelta struct {
	FromMode       string `json:"from_mode"`
	ToMode         string `json:"to_mode"`
	Metric         string `json:"metric"`
	BaseValue      float64 `json:"base_value"`
	NewValue       float64 `json:"new_value"`
	DeltaPercent   float64 `json:"delta_percent"`
	DeltaAbsolute  float64 `json:"delta_absolute"`
	Classification string `json:"classification"` // "strong", "meaningful", "neutral", "regression", "invalid"
}

// ComparativeScorecard represents the full comparative benchmark results
type ComparativeScorecard struct {
	Modes  map[string]*ModeMetrics `json:"modes"`
	Deltas []PairwiseDelta         `json:"deltas"`
	Status string                  `json:"status"`
}

// CalculateComparativeScorecard aggregates results into a comparative report
func CalculateComparativeScorecard(results []EvaluatorResult) ComparativeScorecard {
	scorecard := ComparativeScorecard{
		Modes: make(map[string]*ModeMetrics),
	}

	for i := range results {
		r := &results[i]
		m, ok := scorecard.Modes[r.Mode]
		if !ok {
			m = &ModeMetrics{Mode: r.Mode}
			scorecard.Modes[r.Mode] = m
		}

		m.TotalRuns++
		if r.InputCount > 0 {
			r.IrrelevantRatio = float64(r.InputCount-r.OutputCount) / float64(r.InputCount)
		}
		
		if r.Success {
			m.SuccessfulRuns++
			if !r.FalseSuccessClaim {
				m.TrueSuccessfulRuns++
			}
			m.TokensPerSuccessValues = append(m.TokensPerSuccessValues, r.TokensUsed)
			m.TotalRepairIterations += r.RepairIterations
		}
		if r.FalseSuccessClaim {
			m.FalseSuccessClaims++
		}
		if r.SemanticHit {
			m.TotalSemanticHits++
		}
		m.TotalTokens += r.TokensUsed
		m.TotalContextTokens += r.ContextTokens
		m.TotalIrrelevantRatio += r.IrrelevantRatio
	}

	// Finalize mode metrics
	for _, m := range scorecard.Modes {
		if m.TotalRuns > 0 {
			m.SuccessRate = float64(m.SuccessfulRuns) / float64(m.TotalRuns)
			m.TrueSuccessRate = float64(m.TrueSuccessfulRuns) / float64(m.TotalRuns)
			m.FalseSuccessRate = float64(m.FalseSuccessClaims) / float64(m.TotalRuns)
			m.AvgContextTokens = float64(m.TotalContextTokens) / float64(m.TotalRuns)
			m.AvgIrrelevantRatio = m.TotalIrrelevantRatio / float64(m.TotalRuns)
			m.SemanticHitRate = float64(m.TotalSemanticHits) / float64(m.TotalRuns)
		}
		if m.SuccessfulRuns > 0 {
			m.TokensPerSuccess = float64(m.TotalTokens) / float64(m.SuccessfulRuns)
			m.AvgRetriesPerSuccess = float64(m.TotalRepairIterations) / float64(m.SuccessfulRuns)
			
			// Calculate StdDev
			var sumSqDiff float64
			for _, v := range m.TokensPerSuccessValues {
				diff := float64(v) - m.TokensPerSuccess
				sumSqDiff += diff * diff
			}
			m.StdDevTokensPerSuccess = math.Sqrt(sumSqDiff / float64(m.SuccessfulRuns))
		}
	}

	// Compute pairwise deltas
	scorecard.computeDeltas()
	
	// Set Overall Status
	scorecard.Status = "PASS"
	for _, r := range results {
		isRalphMode := strings.Contains(r.Mode, "Ralph")
		if isRalphMode && r.FalseSuccessClaim {
			scorecard.Status = "FAIL"
		}
		if r.AuthorityOverride {
			scorecard.Status = "FAIL"
		}
	}

	return scorecard
}

func (s *ComparativeScorecard) computeDeltas() {
	pairs := [][]string{
		{"baseline", "tinyMem core"},
		{"tinyMem core", "tinyMem + CoVe"},
		{"tinyMem core", "tinyMem + Ralph"},
		{"tinyMem + CoVe + Ralph", "tinyMem + CoVe + Ralph + Semantic"},
	}

	for _, p := range pairs {
		from, ok1 := s.Modes[p[0]]
		to, ok2 := s.Modes[p[1]]
		if !ok1 || !ok2 {
			continue
		}

		// Use TrueSuccessRate for adjusted comparison
		baseRate := from.SuccessRate
		if from.FalseSuccessRate > 0 {
			baseRate = from.TrueSuccessRate
		}
		newRate := to.SuccessRate
		if to.FalseSuccessRate > 0 {
			newRate = to.TrueSuccessRate
		}

		s.Deltas = append(s.Deltas, calculateDelta(p[0], p[1], "TokensPerSuccess", from.TokensPerSuccess, to.TokensPerSuccess, true, to.SuccessfulRuns))
		s.Deltas = append(s.Deltas, calculateDelta(p[0], p[1], "SuccessRate", baseRate, newRate, false, to.SuccessfulRuns))
		s.Deltas = append(s.Deltas, calculateDelta(p[0], p[1], "FalseSuccessRate", from.FalseSuccessRate, to.FalseSuccessRate, true, to.SuccessfulRuns))
		s.Deltas = append(s.Deltas, calculateDelta(p[0], p[1], "AvgRetriesPerSuccess", from.AvgRetriesPerSuccess, to.AvgRetriesPerSuccess, true, to.SuccessfulRuns))
		s.Deltas = append(s.Deltas, calculateDelta(p[0], p[1], "ContextTokens", from.AvgContextTokens, to.AvgContextTokens, true, to.SuccessfulRuns))
	}
}

func calculateDelta(from, to, metric string, base, newVal float64, lowerIsBetter bool, successfulRuns int) PairwiseDelta {
	d := PairwiseDelta{
		FromMode:      from,
		ToMode:        to,
		Metric:        metric,
		BaseValue:     base,
		NewValue:      newVal,
		DeltaAbsolute: newVal - base,
	}
	
	if successfulRuns == 0 {
		d.Classification = "invalid"
		return d
	}

	if base != 0 {
		d.DeltaPercent = (newVal - base) / base
	}

	improvement := d.DeltaAbsolute
	if lowerIsBetter {
		improvement = -d.DeltaAbsolute
	}

	improvementPercent := d.DeltaPercent
	if lowerIsBetter {
		improvementPercent = -d.DeltaPercent
	}

	d.Classification = "neutral"
	
	switch metric {
	case "TokensPerSuccess", "ContextTokens":
		if improvementPercent >= 0.2 {
			d.Classification = "strong"
		} else if improvementPercent >= 0.1 {
			d.Classification = "meaningful"
		} else if improvementPercent < -0.05 {
			d.Classification = "regression"
		}
	case "SuccessRate":
		if d.DeltaAbsolute >= 0.1 {
			d.Classification = "strong"
		} else if d.DeltaAbsolute >= 0.05 {
			d.Classification = "meaningful"
		} else if d.DeltaAbsolute < -0.05 {
			d.Classification = "regression"
		}
	case "FalseSuccessRate":
		if newVal == 0 && base > 0 {
			d.Classification = "strong"
		} else if improvement > 0 {
			d.Classification = "meaningful"
		}
	case "AvgRetriesPerSuccess":
		if improvement >= 1.0 {
			d.Classification = "strong"
		} else if improvement >= 0.5 {
			d.Classification = "meaningful"
		}
	}

	return d
}


