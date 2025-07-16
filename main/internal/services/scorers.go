package services

import (
	"math"
	"strings"

	"pairingSystem/internal/models"
)

// DefaultStakeScorer implements StakeScorer interface
type DefaultStakeScorer struct {
	weight float64
}

func NewDefaultStakeScorer(weight float64) *DefaultStakeScorer {
	return &DefaultStakeScorer{weight: weight}
}

func (s *DefaultStakeScorer) CalculateStakeScore(provider *models.Provider, policy *models.ConsumerPolicy) (float64, error) {
	if provider.Stake < policy.MinStake {
		return 0.0, nil
	}

	// Logarithmic scaling for stake to prevent extremely high stakes from dominating
	score := math.Log10(float64(provider.Stake)) / math.Log10(10000000) // Scale by 10M
	if score > 1.0 {
		score = 1.0
	}

	return score, nil
}

func (s *DefaultStakeScorer) GetWeight() float64 {
	return s.weight
}

func (s *DefaultStakeScorer) SetWeight(weight float64) {
	s.weight = weight
}

// DefaultLocationScorer implements LocationScorer interface
type DefaultLocationScorer struct {
	weight float64
}

func NewDefaultLocationScorer(weight float64) *DefaultLocationScorer {
	return &DefaultLocationScorer{weight: weight}
}

func (s *DefaultLocationScorer) CalculateLocationScore(provider *models.Provider, policy *models.ConsumerPolicy) (float64, error) {
	if policy.RequiredLocation == "" {
		return 0.5, nil // Neutral score when no location preference
	}

	if provider.Location == policy.RequiredLocation {
		return 1.0, nil // Perfect match
	}

	// Check for regional proximity (same region prefix)
	if len(provider.Location) >= 2 && len(policy.RequiredLocation) >= 2 {
		if strings.HasPrefix(provider.Location, policy.RequiredLocation[:2]) ||
			strings.HasPrefix(policy.RequiredLocation, provider.Location[:2]) {
			return 0.7, nil // Regional match
		}
	}

	return 0.2, nil // Different location
}

func (s *DefaultLocationScorer) GetWeight() float64 {
	return s.weight
}

func (s *DefaultLocationScorer) SetWeight(weight float64) {
	s.weight = weight
}

// DefaultFeatureScorer implements FeatureScorer interface
type DefaultFeatureScorer struct {
	weight float64
}

func NewDefaultFeatureScorer(weight float64) *DefaultFeatureScorer {
	return &DefaultFeatureScorer{weight: weight}
}

func (s *DefaultFeatureScorer) CalculateFeatureScore(provider *models.Provider, policy *models.ConsumerPolicy) (float64, error) {
	if len(policy.RequiredFeatures) == 0 {
		return 0.5, nil // Neutral score when no features required
	}

	// Create a set of provider features for O(1) lookup
	providerFeatures := make(map[string]bool)
	for _, feature := range provider.Features {
		providerFeatures[strings.ToLower(feature)] = true
	}

	matchedFeatures := 0
	for _, required := range policy.RequiredFeatures {
		if providerFeatures[strings.ToLower(required)] {
			matchedFeatures++
		}
	}

	// Calculate score based on percentage of required features matched
	score := float64(matchedFeatures) / float64(len(policy.RequiredFeatures))

	// Bonus for having extra features
	extraFeatures := len(provider.Features) - len(policy.RequiredFeatures)
	if extraFeatures > 0 {
		bonus := math.Min(float64(extraFeatures)*0.05, 0.2) // Max 20% bonus
		score += bonus
	}

	if score > 1.0 {
		score = 1.0
	}

	return score, nil
}

func (s *DefaultFeatureScorer) GetWeight() float64 {
	return s.weight
}

func (s *DefaultFeatureScorer) SetWeight(weight float64) {
	s.weight = weight
}

// LinearStakeScorer - Alternative implementation
type LinearStakeScorer struct {
	weight float64
}

func NewLinearStakeScorer(weight float64) *LinearStakeScorer {
	return &LinearStakeScorer{weight: weight}
}

func (s *LinearStakeScorer) CalculateStakeScore(provider *models.Provider, policy *models.ConsumerPolicy) (float64, error) {
	if provider.Stake < policy.MinStake {
		return 0.0, nil
	}

	// Linear scaling based on stake
	maxStake := float64(1000000) // 1M as reference max
	score := float64(provider.Stake) / maxStake
	if score > 1.0 {
		score = 1.0
	}

	return score, nil
}

func (s *LinearStakeScorer) GetWeight() float64 {
	return s.weight
}

func (s *LinearStakeScorer) SetWeight(weight float64) {
	s.weight = weight
}

// StrictLocationScorer - Alternative implementation that only gives score for exact matches
type StrictLocationScorer struct {
	weight float64
}

func NewStrictLocationScorer(weight float64) *StrictLocationScorer {
	return &StrictLocationScorer{weight: weight}
}

func (s *StrictLocationScorer) CalculateLocationScore(provider *models.Provider, policy *models.ConsumerPolicy) (float64, error) {
	if policy.RequiredLocation == "" {
		return 1.0, nil // All locations acceptable
	}

	if provider.Location == policy.RequiredLocation {
		return 1.0, nil // Perfect match
	}

	return 0.0, nil // No score for non-exact matches
}

func (s *StrictLocationScorer) GetWeight() float64 {
	return s.weight
}

func (s *StrictLocationScorer) SetWeight(weight float64) {
	s.weight = weight
}
