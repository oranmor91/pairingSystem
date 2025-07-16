package main

type PairingSystem interface {
	FilterProviders(providers []*Provider, policy *ConsumerPolicy) []*Provider

	RankProviders(providers []*Provider, policy *ConsumerPolicy) []*PairingScore

	GetPairingList(providers []*Provider, policy *ConsumerPolicy) ([]*Provider, error)
}
