package services

import (
	"fmt"
	"math"

	"pairingSystem/internal/models"
)

// ScoringSystem (Legacy) - maintained for backward compatibility
type ScoringSystem struct {
	compositeScoringSystem ScoringSystemInterface
}

// NewScoringSystem creates a new scoring system using the default composite implementation
func NewScoringSystem() *ScoringSystem {
	return &ScoringSystem{
		compositeScoringSystem: NewDefaultCompositeScoringSystem(),
	}
}

// NewCustomScoringSystem creates a scoring system with a custom composite implementation
func NewCustomScoringSystem(css ScoringSystemInterface) *ScoringSystem {
	return &ScoringSystem{
		compositeScoringSystem: css,
	}
}

// RankProviders ranks providers based on consumer policy preferences
func (ss *ScoringSystem) RankProviders(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.PairingScore {
	return ss.compositeScoringSystem.RankProviders(providers, policy)
}

// GetTopProviders returns the top N providers based on scoring
func (ss *ScoringSystem) GetTopProviders(providers []*models.Provider, policy *models.ConsumerPolicy, topN int) []*models.PairingScore {
	return ss.compositeScoringSystem.GetTopProviders(providers, policy, topN)
}

// CalculateScore calculates the overall score for a provider given a policy
func (ss *ScoringSystem) CalculateScore(provider *models.Provider, policy *models.ConsumerPolicy) (*models.PairingScore, error) {
	return ss.compositeScoringSystem.CalculateScore(provider, policy)
}

// SetScorerWeight sets the weight for a specific scorer component
func (ss *ScoringSystem) SetScorerWeight(name string, weight float64) error {
	if css, ok := ss.compositeScoringSystem.(*CompositeScoringSystem); ok {
		return css.SetScorerWeight(name, weight)
	}
	return fmt.Errorf("scoring system does not support weight modification")
}

// GetScorerWeight gets the weight for a specific scorer component
func (ss *ScoringSystem) GetScorerWeight(name string) (float64, error) {
	if css, ok := ss.compositeScoringSystem.(*CompositeScoringSystem); ok {
		return css.GetScorerWeight(name)
	}
	return 0, fmt.Errorf("scoring system does not support weight retrieval")
}

// GetScorer returns a specific scorer by name
func (ss *ScoringSystem) GetScorer(name string) (interface{}, error) {
	return ss.compositeScoringSystem.GetScorer(name)
}

// ListScorers returns all available scorers
func (ss *ScoringSystem) ListScorers() []string {
	return ss.compositeScoringSystem.ListScorers()
}

// AddScorer adds a new scorer to the system
func (ss *ScoringSystem) AddScorer(name string, scorer interface{}) error {
	return ss.compositeScoringSystem.AddScorer(name, scorer)
}

// RemoveScorer removes a scorer from the system
func (ss *ScoringSystem) RemoveScorer(name string) error {
	return ss.compositeScoringSystem.RemoveScorer(name)
}

// GetName returns the name of the scoring strategy
func (ss *ScoringSystem) GetName() string {
	return ss.compositeScoringSystem.GetName()
}

// GetDescription returns a description of the scoring strategy
func (ss *ScoringSystem) GetDescription() string {
	return ss.compositeScoringSystem.GetDescription()
}

// Legacy methods for backward compatibility (deprecated)

// calculateStakeScore - DEPRECATED: Use StakeScorer interface instead
func (ss *ScoringSystem) calculateStakeScore(provider *models.Provider, policy *models.ConsumerPolicy) float64 {
	if scorer, err := ss.GetScorer("stake"); err == nil {
		if stakeScorer, ok := scorer.(StakeScorer); ok {
			if score, err := stakeScorer.CalculateStakeScore(provider, policy); err == nil {
				return score
			}
		}
	}
	// Fallback to legacy implementation
	return ss.legacyCalculateStakeScore(provider, policy)
}

// calculateLocationScore - DEPRECATED: Use LocationScorer interface instead
func (ss *ScoringSystem) calculateLocationScore(provider *models.Provider, policy *models.ConsumerPolicy) float64 {
	if scorer, err := ss.GetScorer("location"); err == nil {
		if locationScorer, ok := scorer.(LocationScorer); ok {
			if score, err := locationScorer.CalculateLocationScore(provider, policy); err == nil {
				return score
			}
		}
	}
	// Fallback to legacy implementation
	return ss.legacyCalculateLocationScore(provider, policy)
}

// calculateFeatureScore - DEPRECATED: Use FeatureScorer interface instead
func (ss *ScoringSystem) calculateFeatureScore(provider *models.Provider, policy *models.ConsumerPolicy) float64 {
	if scorer, err := ss.GetScorer("features"); err == nil {
		if featureScorer, ok := scorer.(FeatureScorer); ok {
			if score, err := featureScorer.CalculateFeatureScore(provider, policy); err == nil {
				return score
			}
		}
	}
	// Fallback to legacy implementation
	return ss.legacyCalculateFeatureScore(provider, policy)
}

// Legacy implementations (for backward compatibility)

func (ss *ScoringSystem) legacyCalculateStakeScore(provider *models.Provider, policy *models.ConsumerPolicy) float64 {
	if provider.Stake < policy.MinStake {
		return 0.0
	}

	// Logarithmic scaling for stake
	score := math.Log10(float64(provider.Stake)) / math.Log10(10000000)
	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (ss *ScoringSystem) legacyCalculateLocationScore(provider *models.Provider, policy *models.ConsumerPolicy) float64 {
	if policy.RequiredLocation == "" {
		return 0.5
	}

	if provider.Location == policy.RequiredLocation {
		return 1.0
	}

	return 0.2
}

func (ss *ScoringSystem) legacyCalculateFeatureScore(provider *models.Provider, policy *models.ConsumerPolicy) float64 {
	if len(policy.RequiredFeatures) == 0 {
		return 0.5
	}

	providerFeatures := make(map[string]bool)
	for _, feature := range provider.Features {
		providerFeatures[feature] = true
	}

	matchedFeatures := 0
	for _, required := range policy.RequiredFeatures {
		if providerFeatures[required] {
			matchedFeatures++
		}
	}

	score := float64(matchedFeatures) / float64(len(policy.RequiredFeatures))
	return score
}
