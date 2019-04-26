//
// Copyright 2019 Insolar Technologies GbH
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

package bus

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/pkg/errors"
)

const (
	// TopicOutgoing is topic for external calls
	TopicOutgoing = "TopicOutgoing"

	// TopicIncoming is topic for incoming calls
	TopicIncoming = "TopicIncoming"

	// MetaPulse is key for Pulse
	MetaPulse = "pulse"

	// MetaType is key for Type
	MetaType = "type"

	// MetaReceiver is key for Receiver
	MetaReceiver = "receiver"

	// MetaSender is key for Sender
	MetaSender = "sender"
)

//go:generate minimock -i github.com/insolar/insolar/insolar/bus.Sender -o ./ -s _mock.go

// Sender interface sends messages by watermill.
type Sender interface {
	// Send an `Message` and get a `Reply` or error from remote host.
	Send(ctx context.Context, msg *message.Message) <-chan *message.Message
}

type lockedReply struct {
	mutex    sync.RWMutex
	messages chan *message.Message

	isDone uint32
	done   chan struct{}
}

func (r *lockedReply) close() bool {
	if atomic.CompareAndSwapUint32(&r.isDone, 0, 1) {
		close(r.done)
		return true
	}
	return false
}

// Bus is component that sends messages and gives access to replies for them.
type Bus struct {
	pub     message.Publisher
	timeout time.Duration

	repliesMutex sync.RWMutex
	replies      map[string]*lockedReply
}

// NewBus creates Bus instance with provided values.
func NewBus(pub message.Publisher) *Bus {
	return &Bus{
		timeout: time.Minute * 10,
		pub:     pub,
		replies: make(map[string]*lockedReply),
	}
}

func (b *Bus) removeReplyChannel(ctx context.Context, id string) {
	b.repliesMutex.Lock()
	defer b.repliesMutex.Unlock()
	ch, ok := b.replies[id]
	if !ok {
		return
	}

	ch.mutex.Lock()
	inslogger.FromContext(ctx).Infof("close reply channel for message with correlationID %s", id)
	close(ch.messages)
	ch.mutex.Unlock()

	delete(b.replies, id)
}

// Send a watermill's Message and return channel for replies.
func (b *Bus) Send(ctx context.Context, msg *message.Message) (<-chan *message.Message, func()) {
	id := watermill.NewUUID()
	middleware.SetCorrelationID(id, msg)
	reply := &lockedReply{
		messages: make(chan *message.Message),
		done:     make(chan struct{}),
	}
	b.repliesMutex.Lock()
	defer b.repliesMutex.Unlock()

	err := b.pub.Publish(TopicOutgoing, msg)
	if err != nil {
		inslogger.FromContext(ctx).Errorf("can't publish message to %s topic: %s", TopicOutgoing, err.Error())
		return nil, nil
	}

	b.replies[id] = reply

	c := func(b *Bus, reply *lockedReply) func() {
		return func() {

			closed := reply.close()
			if closed {
				b.removeReplyChannel(ctx, id)
			}
		}
	}(b, reply)

	go func(c func()) {
		select {
		case <-reply.done:
			inslogger.FromContext(msg.Context()).Infof("reply channel for message with correlationID %s was closed", id)
		case <-time.After(b.timeout):
			c()
		}
	}(c)

	return reply.messages, c
}

// IncomingMessageRouter is watermill middleware for incoming messages - it decides, how to handle it.
func (b *Bus) IncomingMessageRouter(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		id := middleware.MessageCorrelationID(msg)

		b.repliesMutex.RLock()
		reply, ok := b.replies[id]
		if !ok {
			b.repliesMutex.RUnlock()
			return h(msg)
		}

		reply.mutex.RLock()
		defer reply.mutex.RUnlock()

		b.repliesMutex.RUnlock()

		select {
		case reply.messages <- msg:
			inslogger.FromContext(msg.Context()).Infof("result for message with correlationID %s was send", id)
			return nil, nil
		case <-reply.done:
			return nil, errors.Errorf("can't return result for message with correlationID %s: timeout for reading (%s) was exceeded", id, b.timeout)
		}
	}
}
