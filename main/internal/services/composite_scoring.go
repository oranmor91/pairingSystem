package services

import (
	"fmt"
	"sort"
	"sync"

	"pairingSystem/internal/models"
)

// CompositeScoringSystem implements ScoringSystemInterface
type CompositeScoringSystem struct {
	name        string
	description string
	scorers     map[string]interface{}
	mu          sync.RWMutex
}

// NewCompositeScoringSystem creates a new composite scoring system
func NewCompositeScoringSystem(name, description string) *CompositeScoringSystem {
	return &CompositeScoringSystem{
		name:        name,
		description: description,
		scorers:     make(map[string]interface{}),
	}
}

// NewDefaultCompositeScoringSystem creates a composite scoring system with default scorers
func NewDefaultCompositeScoringSystem() *CompositeScoringSystem {
	css := NewCompositeScoringSystem("Default Composite Scoring", "Uses stake, location, and feature scorers with default weights")

	// Add default scorers
	css.AddScorer("stake", NewDefaultStakeScorer(0.4))
	css.AddScorer("location", NewDefaultLocationScorer(0.3))
	css.AddScorer("features", NewDefaultFeatureScorer(0.3))

	return css
}

// GetName returns the name of the scoring strategy
func (css *CompositeScoringSystem) GetName() string {
	return css.name
}

// GetDescription returns a description of the scoring strategy
func (css *CompositeScoringSystem) GetDescription() string {
	return css.description
}

// AddScorer adds a scorer component to the composite
func (css *CompositeScoringSystem) AddScorer(name string, scorer interface{}) error {
	css.mu.Lock()
	defer css.mu.Unlock()

	// Validate scorer type
	switch scorer.(type) {
	case StakeScorer, LocationScorer, FeatureScorer:
		css.scorers[name] = scorer
		return nil
	default:
		return fmt.Errorf("unsupported scorer type: %T", scorer)
	}
}

// RemoveScorer removes a scorer component from the composite
func (css *CompositeScoringSystem) RemoveScorer(name string) error {
	css.mu.Lock()
	defer css.mu.Unlock()

	if _, exists := css.scorers[name]; !exists {
		return fmt.Errorf("scorer %s not found", name)
	}

	delete(css.scorers, name)
	return nil
}

// GetScorer returns a specific scorer by name
func (css *CompositeScoringSystem) GetScorer(name string) (interface{}, error) {
	css.mu.RLock()
	defer css.mu.RUnlock()

	scorer, exists := css.scorers[name]
	if !exists {
		return nil, fmt.Errorf("scorer %s not found", name)
	}

	return scorer, nil
}

// ListScorers returns all available scorers
func (css *CompositeScoringSystem) ListScorers() []string {
	css.mu.RLock()
	defer css.mu.RUnlock()

	names := make([]string, 0, len(css.scorers))
	for name := range css.scorers {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// CalculateScore calculates the overall score for a provider given a policy
func (css *CompositeScoringSystem) CalculateScore(provider *models.Provider, policy *models.ConsumerPolicy) (*models.PairingScore, error) {
	css.mu.RLock()
	defer css.mu.RUnlock()

	var totalScore float64
	var totalWeight float64
	components := make(map[string]float64)

	// Calculate score from each scorer
	for name, scorer := range css.scorers {
		var score float64
		var weight float64
		var err error

		switch s := scorer.(type) {
		case StakeScorer:
			score, err = s.CalculateStakeScore(provider, policy)
			weight = s.GetWeight()
		case LocationScorer:
			score, err = s.CalculateLocationScore(provider, policy)
			weight = s.GetWeight()
		case FeatureScorer:
			score, err = s.CalculateFeatureScore(provider, policy)
			weight = s.GetWeight()
		default:
			return nil, fmt.Errorf("unsupported scorer type: %T", scorer)
		}

		if err != nil {
			return nil, fmt.Errorf("error calculating score for %s: %w", name, err)
		}

		weightedScore := score * weight
		totalScore += weightedScore
		totalWeight += weight
		components[name] = score
	}

	// Normalize score by total weight
	if totalWeight == 0 {
		return &models.PairingScore{
			Provider:   provider,
			Score:      0.0,
			Components: components,
		}, nil
	}

	normalizedScore := totalScore / totalWeight

	return &models.PairingScore{
		Provider:   provider,
		Score:      normalizedScore,
		Components: components,
	}, nil
}

// RankProviders ranks a list of providers based on the policy
func (css *CompositeScoringSystem) RankProviders(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.PairingScore {
	scores := make([]*models.PairingScore, 0, len(providers))

	for _, provider := range providers {
		score, err := css.CalculateScore(provider, policy)
		if err != nil {
			// Log error but continue with other providers
			continue
		}
		scores = append(scores, score)
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	return scores
}

// GetTopProviders returns the top N providers based on scoring
func (css *CompositeScoringSystem) GetTopProviders(providers []*models.Provider, policy *models.ConsumerPolicy, topN int) []*models.PairingScore {
	scores := css.RankProviders(providers, policy)

	if topN > len(scores) {
		topN = len(scores)
	}

	return scores[:topN]
}

// SetScorerWeight sets the weight for a specific scorer
func (css *CompositeScoringSystem) SetScorerWeight(name string, weight float64) error {
	css.mu.Lock()
	defer css.mu.Unlock()

	scorer, exists := css.scorers[name]
	if !exists {
		return fmt.Errorf("scorer %s not found", name)
	}

	switch s := scorer.(type) {
	case StakeScorer:
		s.SetWeight(weight)
	case LocationScorer:
		s.SetWeight(weight)
	case FeatureScorer:
		s.SetWeight(weight)
	default:
		return fmt.Errorf("unsupported scorer type: %T", scorer)
	}

	return nil
}

// GetScorerWeight gets the weight for a specific scorer
func (css *CompositeScoringSystem) GetScorerWeight(name string) (float64, error) {
	css.mu.RLock()
	defer css.mu.RUnlock()

	scorer, exists := css.scorers[name]
	if !exists {
		return 0, fmt.Errorf("scorer %s not found", name)
	}

	switch s := scorer.(type) {
	case StakeScorer:
		return s.GetWeight(), nil
	case LocationScorer:
		return s.GetWeight(), nil
	case FeatureScorer:
		return s.GetWeight(), nil
	default:
		return 0, fmt.Errorf("unsupported scorer type: %T", scorer)
	}
}
