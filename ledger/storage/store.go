/*
 *    Copyright 2018 INS Ecosystem
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package storage

import (
	"path/filepath"
	"sync"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"

	"github.com/insolar/insolar/ledger/hash"
	"github.com/insolar/insolar/ledger/index"
	"github.com/insolar/insolar/ledger/jetdrop"
	"github.com/insolar/insolar/ledger/record"
)

const (
	scopeIDLifeline byte = 1
	scopeIDRecord   byte = 2
	scopeIDJetDrop  byte = 3
	scopeIDEntropy  byte = 4
)

// Store represents BadgerDB storage implementation.
type Store struct {
	db           *badger.DB
	currentPulse record.PulseNum

	dropWG sync.WaitGroup
}

func setOptions(o *badger.Options) *badger.Options {
	newo := &badger.Options{}
	if o != nil {
		*newo = *o
	} else {
		*newo = badger.DefaultOptions
	}
	return newo
}

// NewStore returns storage.Store with BadgerDB instance initialized by opts.
// Creates database in provided dir or in current directory if dir parameter is empty.
func NewStore(dir string, opts *badger.Options) (*Store, error) {
	opts = setOptions(opts)
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	opts.Dir = dir
	opts.ValueDir = dir

	db, err := badger.Open(*opts)
	if err != nil {
		return nil, errors.Wrap(err, "local database open failed")
	}

	bl := &Store{
		db: db,
	}

	return bl, nil
}

// Close wraps BadgerDB Close method.
//
// From https://godoc.org/github.com/dgraph-io/badger#DB.Close:
// «It's crucial to call it to ensure all the pending updates make their way to disk.
// Calling DB.Close() multiple times is not safe and wouldcause panic.»
func (s *Store) Close() error {
	// TODO: add close flag and mutex guard on Close method
	return s.db.Close()
}

// Get gets value by key in BadgerDB storage.
func (s *Store) Get(key []byte) ([]byte, error) {
	var buf []byte
	txerr := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}
		buf, err = item.ValueCopy(nil)
		return err
	})
	if txerr != nil {
		buf = nil
	}
	return buf, txerr
}

// Set stores value by key.
func (s *Store) Set(key, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

// GetRecord wraps matching transaction manager method.
func (s *Store) GetRecord(ref *record.Reference) (record.Record, error) {
	tx := s.BeginTransaction(false)
	defer tx.Discard()
	rec, err := tx.GetRecord(ref)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// SetRecord wraps matching transaction manager method.
func (s *Store) SetRecord(rec record.Record) (*record.Reference, error) {
	tx := s.BeginTransaction(true)
	defer tx.Discard()

	ref, err := tx.SetRecord(rec)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return ref, nil
}

// GetClassIndex wraps matching transaction manager method.
func (s *Store) GetClassIndex(ref *record.Reference) (*index.ClassLifeline, error) {
	tx := s.BeginTransaction(false)
	defer tx.Discard()

	idx, err := tx.GetClassIndex(ref)
	if err != nil {
		return nil, err
	}
	return idx, nil
}

// SetClassIndex wraps matching transaction manager method.
func (s *Store) SetClassIndex(ref *record.Reference, idx *index.ClassLifeline) error {
	tx := s.BeginTransaction(true)
	defer tx.Discard()

	err := tx.SetClassIndex(ref, idx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// GetObjectIndex wraps matching transaction manager method.
func (s *Store) GetObjectIndex(ref *record.Reference) (*index.ObjectLifeline, error) {
	tx := s.BeginTransaction(false)
	defer tx.Discard()

	idx, err := tx.GetObjectIndex(ref)
	if err != nil {
		return nil, err
	}
	return idx, nil
}

// SetObjectIndex wraps matching transaction manager method.
func (s *Store) SetObjectIndex(ref *record.Reference, idx *index.ObjectLifeline) error {
	tx := s.BeginTransaction(true)
	defer tx.Discard()

	err := tx.SetObjectIndex(ref, idx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// GetDrop returns jet drop for a given pulse number.
func (s *Store) GetDrop(pulse record.PulseNum) (*jetdrop.JetDrop, error) {
	k := prefixkey(scopeIDJetDrop, record.EncodePulseNum(pulse))
	buf, err := s.Get(k)
	if err != nil {
		return nil, err
	}
	drop, err := jetdrop.Decode(buf)
	if err != nil {
		return nil, err
	}
	return drop, nil
}

func (s *Store) waitinflight() {
	s.dropWG.Wait()
}

// SetDrop stores jet drop for given pulse number.
//
// Previous JetDrop should be provided. On success returns saved drop hash.
func (s *Store) SetDrop(pulse record.PulseNum, prevdrop *jetdrop.JetDrop) (*jetdrop.JetDrop, error) {
	s.waitinflight()

	hw := hash.NewSHA3()
	err := s.ProcessSlotHashes(pulse, func(it HashIterator) error {
		for i := 1; it.Next(); i++ {
			b := it.ShallowHash()
			_, err := hw.Write(b)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	drophash := hw.Sum(nil)

	drop := &jetdrop.JetDrop{
		Pulse:    pulse,
		PrevHash: prevdrop.Hash,
		Hash:     drophash,
	}
	encoded, err := jetdrop.Encode(drop)
	if err != nil {
		return nil, err
	}

	k := prefixkey(scopeIDJetDrop, record.EncodePulseNum(pulse))
	err = s.Set(k, encoded)
	if err != nil {
		drop = nil
	}
	return drop, err
}

// GetEntropy wraps matching transaction manager method.
func (s *Store) GetEntropy(pulse record.PulseNum) ([]byte, error) {
	tx := s.BeginTransaction(false)
	defer tx.Discard()

	idx, err := tx.GetEntropy(pulse)
	if err != nil {
		return nil, err
	}
	return idx, nil
}

// SetEntropy wraps matching transaction manager method.
func (s *Store) SetEntropy(pulse record.PulseNum, entropy []byte) error {
	tx := s.BeginTransaction(true)
	defer tx.Discard()

	err := tx.SetEntropy(pulse, entropy)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// SetCurrentPulse sets current pulse number.
func (s *Store) SetCurrentPulse(pulse record.PulseNum) {
	s.currentPulse = pulse
}

// GetCurrentPulse returns current pulse number.
func (s *Store) GetCurrentPulse() record.PulseNum {
	return s.currentPulse
}

// BeginTransaction opens a new transaction.
// All methods called on returned transaction manager will persist changes
// only after success on "Commit" call.
func (s *Store) BeginTransaction(update bool) *TransactionManager {
	if update {
		s.dropWG.Add(1)
	}
	return &TransactionManager{
		store:  s,
		txn:    s.db.NewTransaction(update),
		update: update,
	}
}

// View accepts transaction function. All calls to received transaction manager will be consistent.
func (s *Store) View(fn func(*TransactionManager) error) error {
	tx := s.BeginTransaction(false)
	defer tx.Discard()
	return fn(tx)
}

// Update accepts transaction function and commits changes. All calls to received transaction manager will be
// consistent and written tp disk or an error will be returned.
func (s *Store) Update(fn func(*TransactionManager) error) error {
	tx := s.BeginTransaction(true)
	defer tx.Discard()

	err := fn(tx)
	if err != nil {
		return err
	}
	return tx.Commit()
}
