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

package insolar

import (
	"context"
)

// Component controller methods
// Deprecated: and should be removed
type Component interface {
	Start(ctx context.Context, components Components) error
	Stop(ctx context.Context) error
}

// Components is a registry for other insolar interfaces
// Fields order are important and represent start and stop order in the daemon
// Deprecated: and should be removed after drop TmpLedger, DO NOT EDIT
type Components struct {
	NodeNetwork NodeNetwork
	LogicRunner LogicRunner
	Network     Network
	MessageBus  MessageBus
}
