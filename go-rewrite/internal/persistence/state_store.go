package persistence

import "meridian-go-rewrite/internal/solana/types"

type StateStore struct {
	store *Store[types.StateData]
}

func NewStateStore(path string) (*StateStore, error) {
	initial := types.StateData{Positions: make(map[string]types.PositionState)}
	s, err := NewStore(path, initial)
	if err != nil {
		return nil, err
	}
	return &StateStore{store: s}, nil
}

func (ss *StateStore) GetPositions() map[string]types.PositionState {
	return ss.store.Read().Positions
}

func (ss *StateStore) UpdatePosition(id string, pos types.PositionState) error {
	return ss.store.Update(func(data *types.StateData) error {
		data.Positions[id] = pos
		return nil
	})
}
