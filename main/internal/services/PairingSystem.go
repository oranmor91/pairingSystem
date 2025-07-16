package services

import (
	"pairingSystem/internal/models"
)

type PairingSystem interface {
	FilterProviders(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.Provider

	RankProviders(providers []*models.Provider, policy *models.ConsumerPolicy) []*models.PairingScore

	GetPairingList(providers []*models.Provider, policy *models.ConsumerPolicy) ([]*models.Provider, error)
}
