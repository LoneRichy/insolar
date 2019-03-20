//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package jet

import (
	"context"
	"sync"

	"github.com/insolar/insolar/core"
)

var (
	_ Accessor = &Store{}
	_ Modifier = &Store{}
)

type lockedTree struct {
	sync.RWMutex
	t *Tree
}

func (lt *lockedTree) find(recordID core.RecordID) (core.JetID, bool) {
	lt.RLock()
	defer lt.RUnlock()
	return lt.t.Find(recordID)
}

func (lt *lockedTree) update(id core.JetID, setActual bool) {
	lt.Lock()
	defer lt.Unlock()
	lt.t.Update(id, setActual)
}

func (lt *lockedTree) leafIDs() []core.JetID {
	lt.RLock()
	defer lt.RUnlock()
	return lt.t.LeafIDs()
}

func (lt *lockedTree) clone(keep bool) *Tree {
	lt.RLock()
	defer lt.RUnlock()
	return lt.t.Clone(keep)
}

func (lt *lockedTree) split(id core.JetID) (core.JetID, core.JetID, error) {
	lt.RLock()
	defer lt.RUnlock()
	return lt.t.Split(id)
}

// Store stores jet trees per pulse.
// It provides methods for querying and modification this trees.
type Store struct {
	sync.RWMutex
	trees map[core.PulseNumber]*lockedTree
}

// NewStore creates new Store instance.
func NewStore() *Store {
	return &Store{
		trees: map[core.PulseNumber]*lockedTree{},
	}
}

// All returns all jet from jet tree for provided pulse.
func (s *Store) All(ctx context.Context, pulse core.PulseNumber) []core.JetID {
	return s.ltreeForPulse(pulse).leafIDs()
}

// ForID finds jet in jet tree for provided pulse and object.
// Always returns jet id and activity flag for this jet.
func (s *Store) ForID(ctx context.Context, pulse core.PulseNumber, recordID core.RecordID) (core.JetID, bool) {
	return s.ltreeForPulse(pulse).find(recordID)
}

// Update updates jet tree for specified pulse.
func (s *Store) Update(ctx context.Context, pulse core.PulseNumber, setActual bool, ids ...core.JetID) {
	s.Lock()
	defer s.Unlock()

	ltree := s.ltreeForPulseUnsafe(pulse)
	for _, id := range ids {
		ltree.update(id, setActual)
	}
	// required because TreeForPulse could return new tree.
	s.trees[pulse] = ltree
}

// Split performs jet split and returns resulting jet ids.
func (s *Store) Split(
	ctx context.Context, pulse core.PulseNumber, id core.JetID,
) (core.JetID, core.JetID, error) {
	ltree := s.ltreeForPulse(pulse)
	left, right, err := ltree.split(id)
	if err != nil {
		return core.ZeroJetID, core.ZeroJetID, err
	}
	return left, right, nil
}

// Clone copies tree from one pulse to another. Use it to copy past tree into new pulse.
func (s *Store) Clone(
	ctx context.Context, from, to core.PulseNumber,
) {
	newTree := s.ltreeForPulse(from).clone(false)

	s.Lock()
	s.trees[to] = &lockedTree{
		t: newTree,
	}
	s.Unlock()
}

// Delete concurrent safe.
func (s *Store) Delete(
	ctx context.Context, pulse core.PulseNumber,
) {
	s.Lock()
	defer s.Unlock()
	delete(s.trees, pulse)
}

// ltreeForPulse returns jet tree with lock for pulse, it's concurrent safe.
func (s *Store) ltreeForPulse(pulse core.PulseNumber) *lockedTree {
	s.Lock()
	defer s.Unlock()
	return s.ltreeForPulseUnsafe(pulse)
}

// ltreeForPulseUnsafe returns jet tree with lock for pulse, it's concurrent unsafe and requires write lock.
func (s *Store) ltreeForPulseUnsafe(pulse core.PulseNumber) *lockedTree {
	if ltree, ok := s.trees[pulse]; ok {
		return ltree
	}

	ltree := &lockedTree{
		t: NewTree(pulse == core.GenesisPulse.PulseNumber),
	}
	s.trees[pulse] = ltree
	return ltree
}