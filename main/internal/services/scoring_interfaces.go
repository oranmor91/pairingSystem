package services

import "pairingSystem/internal/models"

// ScoringStrategy defines the interface for different scoring strategies
type ScoringStrategy interface {
	// CalculateScore calculates the overall score for a provider given a policy
	CalculateScore(provider *models.Provider, policy *models.ConsumerPolicy) (*models.PairingScore, error)
	// GetName returns the name of the scoring strategy
	GetName() string
	// GetDescription returns a description of the scoring strategy
	GetDescription() string
}

// StakeScorer defines the interface for stake-based scoring
type StakeScorer interface {
	// CalculateStakeScore calculates the stake component of the score
	CalculateStakeScore(provider *models.Provider, policy *models.ConsumerPolicy) (float64, error)
	// GetWeight returns the weight of this scorer in the overall calculation
	GetWeight() float64
	// SetWeight sets the weight of this scorer
	SetWeight(weight float64)
}

// LocationScorer defines the interface for location-based scoring
type LocationScorer interface {
	// CalculateLocationScore calculates the location component of the score
	CalculateLocationScore(provider *models.Provider, policy *models.ConsumerPolicy) (float64, error)
	// GetWeight returns the weight of this scorer in the overall calculation
	GetWeight() float64
	// SetWeight sets the weight of this scorer
	SetWeight(weight float64)
}

// FeatureScorer defines the interface for feature-based scoring
type FeatureScorer interface {
	// CalculateFeatureScore calculates the feature component of the score
	CalculateFeatureScore(provider *models.Provider, policy *models.ConsumerPolicy) (float64, error)
	// GetWeight returns the weight of this scorer in the overall calculation
	GetWeight() float64
	// SetWeight sets the weight of this scorer
	SetWeight(weight float64)
}

// CompositeScorer defines the interface for composite scoring that combines multiple scorers
type CompositeScorer interface {
	// AddScorer adds a scorer component to the composite
	AddScorer(name string, scorer interface{}) error
	// RemoveScorer removes a scorer component from the composite
	RemoveScorer(name string) error
	// GetScorer returns a specific scorer by name
	GetScorer(name string) (interface{}, error)
	// ListScorers returns all available scorers
	ListScorers() []string
}

// ScoringSystemInterface defines the main interface for the scoring system
type ScoringSystemInterface interface {
	ScoringStrategy
	CompositeScorer

	// RankProviders ranks a list of providers based on the policy
	RankProviders(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.PairingScore
	// GetTopProviders returns the top N providers based on scoring
	GetTopProviders(providers []*models.Provider, policy *models.ConsumerPolicy, topN int) []*models.PairingScore
}
