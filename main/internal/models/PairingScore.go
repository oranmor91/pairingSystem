package models

type PairingScore struct {
	Provider   *Provider
	Score      float64
	Components map[string]float64
}
