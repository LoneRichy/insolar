/*
 *    Copyright 2019 Insolar
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

package queue

import (
	"sync"
	"sync/atomic"
)

// MutexQueue is mutex-based realization of IQueue
type MutexQueue struct {
	locker sync.Mutex
	head   *QueueItem
}

// NewMutexQueue creates new instance of MutexQueue
func NewMutexQueue() IQueue {
	queue := &MutexQueue{
		head: &emptyQueueItem,
	}
	return queue
}

func (q *MutexQueue) sinkPush(newNode *QueueItem) bool {
	q.locker.Lock()
	defer q.locker.Unlock()

	if q.head == nil {
		return false
	}

	if q.HasSignal() {
		// do smth interesting
	}

	newNode.next = q.head
	newNode.index = q.head.index + 1

	// just max =(
	if q.head.signal > newNode.itemType {
		newNode.signal = q.head.signal
	} else {
		newNode.signal = newNode.itemType
	}

	q.head = newNode

	return true
}

// SinkPush is implementation for IQueue
func (q *MutexQueue) SinkPush(data interface{}) bool {

	newNode := &QueueItem{
		payload: data,
	}

	return q.sinkPush(newNode)
}

// SinkPushAll is implementation for IQueue
func (q *MutexQueue) SinkPushAll(data []interface{}) bool {
	inputSize := len(data)
	lastElement := &QueueItem{}
	newHead := lastElement

	lastElement.payload = data[inputSize-1]
	for i := inputSize - 2; i >= 0; i-- {
		lastElement.next = &QueueItem{}
		lastElement = lastElement.next
		lastElement.payload = data[i]
	}

	q.locker.Lock()
	defer q.locker.Unlock()

	if q.head == nil {
		return false
	}

	if q.HasSignal() {
		// do smth interesting
	}

	nextIndex := q.head.index + 1
	tmpHead := newHead
	for i := inputSize - 1; i >= 0; i-- {
		tmpHead.index = nextIndex + uint(i)
		tmpHead = tmpHead.next
	}

	lastElement.next = q.head
	// lastNew.signal = max(head.signal, lastNew.type) // TODO: ? What is that ?

	q.head = newHead

	return true
}

func (q *MutexQueue) checkAndGetHead() *QueueItem {
	if q.head == nil || q.head == &emptyQueueItem {
		return nil
	}

	return q.head
}

// get pointer to head and unfold linked list to slice:
//  all signals will be at the begging of the slice
func convertSublistToArray(localHead *QueueItem) []interface{} {
	result := make([]interface{}, localHead.index)

	current := localHead
	for i := uint(0); i < localHead.index; i++ {
		result[localHead.index-i-1] = current.payload
		current = current.next
	}

	return result
}

// SinkPushAll is implementation for IQueue
func (q *MutexQueue) RemoveAll() []interface{} {

	var localHead *QueueItem
	q.locker.Lock()
	localHead = q.checkAndGetHead()
	if localHead == nil {
		q.locker.Unlock()
		return []interface{}{}
	}
	q.head = &emptyQueueItem
	q.locker.Unlock()

	return convertSublistToArray(localHead)
}

func (q *MutexQueue) BlockAndRemoveAll() []interface{} {
	var localHead *QueueItem
	q.locker.Lock()
	localHead = q.checkAndGetHead()
	if localHead == nil {
		q.locker.Unlock()
		return []interface{}{}
	}
	q.head = nil
	q.locker.Unlock()

	return convertSublistToArray(localHead)
}

// SinkPushAll is implementation for IQueue
func (q *MutexQueue) Unblock() bool {
	q.locker.Lock()
	defer q.locker.Unlock()

	if q.head != nil {
		return false
	}

	q.head = &emptyQueueItem
	return true
}

// SinkPushAll is implementation for IQueue
func (q *MutexQueue) PushSignal(signalType uint32, callback SyncDone) bool {
	newNode := &QueueItem{
		payload: callback,
		signal:  signalType,
	}

	return q.sinkPush(newNode)
}

// SinkPushAll is implementation for IQueue
func (q *MutexQueue) HasSignal() bool {
	return atomic.LoadUint32(&q.head.signal) != 0
}
