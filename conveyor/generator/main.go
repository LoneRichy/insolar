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

package main

import (
	"github.com/insolar/insolar/conveyor/generator/generator"
	"github.com/insolar/insolar/conveyor/generator/state_machines/get_object"
	"github.com/insolar/insolar/conveyor/generator/state_machines/initial"
	"github.com/insolar/insolar/conveyor/generator/state_machines/sample"
)

func main() {
	gen := generator.NewGenerator()
	getobject.Register(gen)
	sample.Register(gen)
	initial.Register(gen)
	gen.CheckAllMachines()
	gen.GenerateStateMachines()
	gen.GenerateMatrix()
}