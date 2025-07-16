package main

import (
	"fmt"
	"math"
	"sort"
)

// ScoreComponent represents a component of the scoring system
type ScoreComponent struct {
	Name   string
	Weight float64
	Score  float64
}

// ScoringSystem handles the ranking and scoring of providers
type ScoringSystem struct {
	// Component weights for scoring
	stakeWeight    float64
	locationWeight float64
	featureWeight  float64

	// Normalization parameters
	maxStake    int64
	minStake    int64
	initialized bool
}

// NewScoringSystem creates a new scoring system with default weights
func NewScoringSystem() *ScoringSystem {
	return &ScoringSystem{
		stakeWeight:    0.5, // 50% weight for stake
		locationWeight: 0.2, // 20% weight for location match
		featureWeight:  0.3, // 30% weight for feature completeness
		initialized:    false,
	}
}

// SetWeights allows customization of scoring component weights
func (ss *ScoringSystem) SetWeights(stake, location, feature float64) error {
	total := stake + location + feature
	if math.Abs(total-1.0) > 0.001 {
		return fmt.Errorf("weights must sum to 1.0, got %f", total)
	}

	ss.stakeWeight = stake
	ss.locationWeight = location
	ss.featureWeight = feature
	return nil
}

// initializeNormalization calculates min/max values for normalization
func (ss *ScoringSystem) initializeNormalization(providers []*Provider) {
	if len(providers) == 0 {
		ss.maxStake = 1000000 // Default max stake
		ss.minStake = 0
		ss.initialized = true
		return
	}

	ss.maxStake = providers[0].Stake
	ss.minStake = providers[0].Stake

	for _, provider := range providers {
		if provider.Stake > ss.maxStake {
			ss.maxStake = provider.Stake
		}
		if provider.Stake < ss.minStake {
			ss.minStake = provider.Stake
		}
	}

	// Ensure we have a valid range
	if ss.maxStake == ss.minStake {
		ss.maxStake = ss.minStake + 1
	}

	ss.initialized = true
}

// calculateStakeScore calculates normalized stake score (0-1)
func (ss *ScoringSystem) calculateStakeScore(provider *Provider) float64 {
	if !ss.initialized {
		return 0.5 // Default score if not initialized
	}

	stakeRange := ss.maxStake - ss.minStake
	if stakeRange == 0 {
		return 1.0
	}

	// Normalize to 0-1 range
	normalizedScore := float64(provider.Stake-ss.minStake) / float64(stakeRange)

	// Ensure bounds
	if normalizedScore < 0 {
		normalizedScore = 0
	}
	if normalizedScore > 1 {
		normalizedScore = 1
	}

	return normalizedScore
}

// calculateLocationScore calculates location match score (0-1)
func (ss *ScoringSystem) calculateLocationScore(provider *Provider, policy *ConsumerPolicy) float64 {
	if policy.RequiredLocation == "" {
		return 1.0 // No location requirement, full score
	}

	if provider.Location == policy.RequiredLocation {
		return 1.0 // Perfect match
	}

	// Could implement geographic proximity scoring here
	// For now, it's binary: match or no match
	return 0.0
}

// calculateFeatureScore calculates feature completeness score (0-1)
func (ss *ScoringSystem) calculateFeatureScore(provider *Provider, policy *ConsumerPolicy) float64 {
	if len(policy.RequiredFeatures) == 0 {
		return 1.0 // No feature requirements, full score
	}

	// Create a set of provider features for O(1) lookup
	providerFeatures := make(map[string]bool)
	for _, feature := range provider.Features {
		providerFeatures[feature] = true
	}

	// Count matching features
	matchedFeatures := 0
	for _, required := range policy.RequiredFeatures {
		if providerFeatures[required] {
			matchedFeatures++
		}
	}

	// Calculate completeness ratio
	score := float64(matchedFeatures) / float64(len(policy.RequiredFeatures))

	// Bonus for having extra features
	extraFeatures := len(provider.Features) - len(policy.RequiredFeatures)
	if extraFeatures > 0 && matchedFeatures == len(policy.RequiredFeatures) {
		// Small bonus for extra features, capped at 0.1
		bonus := math.Min(0.1, float64(extraFeatures)*0.02)
		score = math.Min(1.0, score+bonus)
	}

	return score
}

// ScoreProvider calculates a comprehensive score for a provider
func (ss *ScoringSystem) ScoreProvider(provider *Provider, policy *ConsumerPolicy, allProviders []*Provider) *PairingScore {
	if !ss.initialized {
		ss.initializeNormalization(allProviders)
	}

	// Calculate individual component scores
	stakeScore := ss.calculateStakeScore(provider)
	locationScore := ss.calculateLocationScore(provider, policy)
	featureScore := ss.calculateFeatureScore(provider, policy)

	// Calculate weighted final score
	finalScore := (stakeScore * ss.stakeWeight) +
		(locationScore * ss.locationWeight) +
		(featureScore * ss.featureWeight)

	// Ensure score is in valid range [0, 1]
	finalScore = math.Max(0, math.Min(1, finalScore))

	// Create component breakdown
	components := map[string]float64{
		"stake":    stakeScore,
		"location": locationScore,
		"features": featureScore,
		"final":    finalScore,
	}

	return &PairingScore{
		Provider:   provider,
		Score:      finalScore,
		Components: components,
	}
}

// RankProviders scores and ranks all providers according to the policy
func (ss *ScoringSystem) RankProviders(providers []*Provider, policy *ConsumerPolicy) []*PairingScore {
	if len(providers) == 0 {
		return []*PairingScore{}
	}

	// Initialize normalization if needed
	if !ss.initialized {
		ss.initializeNormalization(providers)
	}

	// Score all providers
	scores := make([]*PairingScore, len(providers))
	for i, provider := range providers {
		scores[i] = ss.ScoreProvider(provider, policy, providers)
	}

	// Sort by score (highest first)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	return scores
}

// GetTopProviders returns the top N providers after ranking
func (ss *ScoringSystem) GetTopProviders(providers []*Provider, policy *ConsumerPolicy, n int) []*Provider {
	scores := ss.RankProviders(providers, policy)

	// Limit to requested number
	if n > len(scores) {
		n = len(scores)
	}

	result := make([]*Provider, n)
	for i := 0; i < n; i++ {
		result[i] = scores[i].Provider
	}

	return result
}

// GetScoringStats returns statistics about the scoring distribution
func (ss *ScoringSystem) GetScoringStats(providers []*Provider, policy *ConsumerPolicy) map[string]interface{} {
	scores := ss.RankProviders(providers, policy)

	if len(scores) == 0 {
		return map[string]interface{}{
			"count": 0,
			"avg":   0.0,
			"min":   0.0,
			"max":   0.0,
		}
	}

	// Calculate statistics
	var sum, min, max float64
	min = scores[0].Score
	max = scores[0].Score

	for _, score := range scores {
		sum += score.Score
		if score.Score < min {
			min = score.Score
		}
		if score.Score > max {
			max = score.Score
		}
	}

	avg := sum / float64(len(scores))

	return map[string]interface{}{
		"count": len(scores),
		"avg":   avg,
		"min":   min,
		"max":   max,
		"range": max - min,
	}
}

// ResetNormalization resets the normalization parameters
func (ss *ScoringSystem) ResetNormalization() {
	ss.initialized = false
	ss.maxStake = 0
	ss.minStake = 0
}
