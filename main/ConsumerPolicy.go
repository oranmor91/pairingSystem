package main

type ConsumerPolicy struct {
	RequiredLocation string
	RequiredFeatures []string
	MinStake         int64
}
