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

package handle

import (
	"context"

	"github.com/insolar/insolar/insolar/flow"
	"github.com/insolar/insolar/insolar/flow/bus"
	"github.com/insolar/insolar/insolar/message"
	"github.com/insolar/insolar/ledger/light/proc"
)

type SetBlob struct {
	dep     *proc.Dependencies
	msg     *message.SetBlob
	replyTo chan<- bus.Reply
}

func NewSetBlob(dep *proc.Dependencies, rep chan<- bus.Reply, msg *message.SetBlob) *SetBlob {
	return &SetBlob{
		dep:     dep,
		msg:     msg,
		replyTo: rep,
	}
}

func (s *SetBlob) Present(ctx context.Context, f flow.Flow) error {
	jet := proc.NewFetchJet(*s.msg.TargetRef.Record(), flow.Pulse(ctx), s.replyTo)
	s.dep.FetchJet(jet)
	if err := f.Procedure(ctx, jet, true); err != nil {
		return err
	}
	hot := proc.NewWaitHot(jet.Result.Jet, flow.Pulse(ctx), s.replyTo)
	s.dep.WaitHot(hot)
	if err := f.Procedure(ctx, hot, true); err != nil {
		return err
	}

	setBlob := proc.NewSetBlob(jet.Result.Jet, s.replyTo, s.msg)
	s.dep.SetBlob(setBlob)
	return f.Procedure(ctx, setBlob, false)
}
