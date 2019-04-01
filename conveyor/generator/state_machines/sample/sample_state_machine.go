/*
 *    Copyright 2019 Insolar Technologies
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

package sample

import (
	"context"

	"github.com/insolar/insolar/conveyor/fsm"
	"github.com/insolar/insolar/conveyor/generator/generator"
)

// custom types
type CustomEvent struct{}
type CustomPayload struct{}
type CustomAdapterHelper interface{}

const (
	InitState fsm.ElementState = iota
	StateFirst
	StateSecond
)

func Register(g *generator.Generator) {
	g.AddMachine("SampleStateMachine").
		InitFuture(initFutureHandler).
		Init(initPresentHandler, StateFirst).
		Transition(StateFirst, transitPresentFirst, StateSecond).
		Transition(StateSecond, transitPresentSecond, 0)
}

func initPresentHandler(ctx context.Context, helper fsm.SlotElementHelper, input CustomEvent, payload interface{}) (fsm.ElementState, *CustomPayload) {
	return StateFirst, nil
}

func initFutureHandler(ctx context.Context, helper fsm.SlotElementHelper, input CustomEvent, payload interface{}) (fsm.ElementState, *CustomPayload) {
	panic("implement me")
}

func transitPresentFirst(ctx context.Context, helper fsm.SlotElementHelper, input CustomEvent, payload *CustomPayload, adapterHelper CustomAdapterHelper) fsm.ElementState {
	return StateSecond
}

func transitPresentSecond(ctx context.Context, helper fsm.SlotElementHelper, input CustomEvent, payload *CustomPayload, adapterHelper CustomAdapterHelper) fsm.ElementState {
	return 0
}