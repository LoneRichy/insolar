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

package pulse

import (
	"context"

	"github.com/insolar/insolar/insolar"
)

type contextKey struct{}

func FromContext(ctx context.Context) insolar.PulseNumber {
	return ctx.Value(contextKey{}).(insolar.PulseNumber)
}

func ContextWith(ctx context.Context, pn insolar.PulseNumber) context.Context {
	return context.WithValue(ctx, contextKey{}, pn)
}

func FromContextString(ctx context.Context) string {
	return ctx.Value(contextKey{}).(string)
}

func ContextWithString(ctx context.Context, pn string) context.Context {
	return context.WithValue(ctx, contextKey{}, pn)
}
