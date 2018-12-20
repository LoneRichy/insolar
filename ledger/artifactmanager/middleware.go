/*
 *    Copyright 2018 Insolar
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

package artifactmanager

import (
	"context"

	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/core/message"
	"github.com/insolar/insolar/core/reply"
	"github.com/insolar/insolar/ledger/storage"
	"github.com/pkg/errors"
)

const (
	fetchJetReties = 10
)

type middleware struct {
	db             *storage.DB
	jetCoordinator core.JetCoordinator
	messageBus     core.MessageBus
}

type jetKey struct{}

func contextWithJet(ctx context.Context, jetID core.RecordID) context.Context {
	return context.WithValue(ctx, jetKey{}, jetID)
}

func jetFromContext(ctx context.Context) core.RecordID {
	val := ctx.Value(jetKey{})
	j, ok := val.(core.RecordID)
	if !ok {
		panic("failed to extract jet from context")
	}

	return j
}

func (m *middleware) checkJet(handler core.MessageHandler) core.MessageHandler {
	return func(ctx context.Context, parcel core.Parcel) (core.Reply, error) {
		genericMsg := parcel.Message()
		msg, ok := genericMsg.(message.LedgerMessage)
		if !ok {
			return nil, errors.New("unexpected message")
		}
		if msg.DefaultTarget() == nil {
			return nil, errors.New("unexpected message")
		}
		targetPulse := msg.TargetPulse()
		if targetPulse == core.PulseNumberCurrent {
			targetPulse = parcel.Pulse()
		}

		// Calculate jet.
		var jetID core.RecordID
		if msg.TargetPulse() == core.PulseNumberJet {
			jetID = *msg.DefaultTarget().Record()
		} else {
			j, err := m.fetchJet(ctx, *msg.DefaultTarget().Record(), targetPulse, parcel.Pulse(), fetchJetReties)
			if err != nil {
				return nil, errors.Wrap(err, "failed to fetch jet tree")
			}
			jetID = *j
		}

		// Check if jet is ours.
		node, err := m.jetCoordinator.LightExecutorForJet(ctx, jetID, targetPulse)
		if err != nil {
			return nil, errors.Wrap(err, "failed to calculate executor for jet")
		}
		if *node != m.jetCoordinator.Me() {
			return &reply.JetMiss{JetID: jetID}, nil
		}

		return handler(contextWithJet(ctx, jetID), parcel)
	}
}

func (m *middleware) saveParcel(handler core.MessageHandler) core.MessageHandler {
	return func(ctx context.Context, parcel core.Parcel) (core.Reply, error) {
		jetID := jetFromContext(ctx)
		pulse, err := m.db.GetLatestPulse(ctx)
		if err != nil {
			return nil, err
		}
		err = m.db.SetMessage(ctx, jetID, pulse.Pulse.PulseNumber, parcel)
		if err != nil {
			return nil, err
		}

		return handler(ctx, parcel)
	}
}

func (m *middleware) fetchJet(
	ctx context.Context, target core.RecordID, parcelPulse, targetPulse core.PulseNumber, retries int,
) (*core.RecordID, error) {
	if retries < 0 {
		return nil, errors.New("retries exceeded")
	}

	// Look in the local tree. Return if the actual jet found.
	tree, err := m.db.GetJetTree(ctx, targetPulse)
	if err != nil {
		return nil, err
	}
	jetID, actual := tree.Find(target)
	if actual {
		return jetID, nil
	}

	// Couldn't find the actual jet locally. Ask for the jet from the previous executor.
	prevPulse, err := m.db.GetPreviousPulse(ctx, parcelPulse)
	if err != nil {
		return nil, err
	}
	prevExecutor, err := m.jetCoordinator.LightExecutorForJet(ctx, *jetID, prevPulse.Pulse.PulseNumber)
	if err != nil {
		return nil, err
	}
	currentPulse, err := m.db.GetLatestPulse(ctx)
	if err != nil {
		return nil, err
	}
	rep, err := m.messageBus.Send(
		ctx,
		&message.GetJet{Object: target},
		currentPulse.Pulse,
		&core.MessageSendOptions{Receiver: prevExecutor},
	)
	r, ok := rep.(*reply.Jet)
	if !ok {
		return nil, ErrUnexpectedReply
	}

	// TODO: check if the same executor again
	// TODO: check if the same jet again

	// Update local tree.
	err = m.db.UpdateJetTree(ctx, targetPulse, r.Actual)
	if err != nil {
		return nil, err
	}

	// Repeat the process again.
	return m.fetchJet(ctx, target, parcelPulse, targetPulse, retries-1)
}
