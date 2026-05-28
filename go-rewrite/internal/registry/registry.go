package registry

import (
	"meridian-go-rewrite/internal/persistence"
	"meridian-go-rewrite/internal/signal"
)

var (
	PoolMemory    *persistence.PoolMemoryStore
	Lessons       *persistence.LessonsStore
	DecisionLog   *persistence.DecisionLogStore
	Strategies    *persistence.StrategyLibraryStore
	TokenBl       *persistence.TokenBlacklistStore
	SmartWallets  *persistence.SmartWalletStore
	State         *persistence.StateStore
	SignalTracker *signal.Tracker
	SignalWeights *signal.Weights
)

func Init(dataDir string) error {
	var err error

	PoolMemory, err = persistence.NewPoolMemoryStore(dataDir + "/pool-memory.json")
	if err != nil {
		return err
	}

	Lessons, err = persistence.NewLessonsStore(dataDir + "/lessons.json")
	if err != nil {
		return err
	}

	DecisionLog, err = persistence.NewDecisionLogStore(dataDir + "/decision-log.json")
	if err != nil {
		return err
	}

	Strategies, err = persistence.NewStrategyLibraryStore(dataDir + "/strategies.json")
	if err != nil {
		return err
	}

	TokenBl, err = persistence.NewTokenBlacklistStore(dataDir + "/token-blacklist.json")
	if err != nil {
		return err
	}

	SmartWallets, err = persistence.NewSmartWalletStore(dataDir + "/smart-wallets.json")
	if err != nil {
		return err
	}

	State, err = persistence.NewStateStore(dataDir + "/state.json")
	if err != nil {
		return err
	}

	SignalTracker = signal.NewTracker()
	SignalWeights, err = signal.NewWeights(dataDir + "/signal-weights.json")
	if err != nil {
		return err
	}

	return nil
}
