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

package rootdomain

import (
	"encoding/json"

	"github.com/insolar/insolar/genesis/proxy/member"
	"github.com/insolar/insolar/genesis/proxy/wallet"

	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/logicrunner/goplugin/foundation"
)

type RootDomain struct {
	foundation.BaseContract
}

func (rd *RootDomain) CreateMember(name string) string {
	memberHolder := member.New(name)
	m := memberHolder.AsChild(rd.GetReference())
	wHolder := wallet.New(1000)
	wHolder.AsDelegate(m.GetReference())
	return m.GetReference().String()
}

func (rd *RootDomain) GetBalance(reference string) uint {
	w := wallet.GetImplementationFrom(core.NewRefFromBase58(reference))
	return w.GetTotalBalance()
}

func (rd *RootDomain) SendMoney(from string, to string, amount uint) bool {
	walletFrom := wallet.GetImplementationFrom(core.NewRefFromBase58(from))

	v := core.NewRefFromBase58(to)
	walletFrom.Transfer(amount, &v)

	return true
}

func (rd *RootDomain) getUserInfoMap(m *member.Member) map[string]interface{} {
	w := wallet.GetImplementationFrom(m.GetReference())
	res := map[string]interface{}{
		"member": m.GetName(),
		"wallet": w.GetTotalBalance(),
	}
	return res
}

func (rd *RootDomain) DumpUserInfo(reference string) []byte {
	m := member.GetObject(core.NewRefFromBase58(reference))
	res := rd.getUserInfoMap(m)
	resJSON, _ := json.Marshal(res)
	return resJSON
}

func (rd *RootDomain) DumpAllUsers() []byte {
	res := []map[string]interface{}{}
	crefs, err := rd.GetChildrenTyped(member.ClassReference)
	if err != nil {
		panic(err)
	}
	for _, cref := range crefs {
		m := member.GetObject(cref)
		userInfo := rd.getUserInfoMap(m)
		res = append(res, userInfo)
	}
	resJSON, _ := json.Marshal(res)
	return resJSON
}

func NewRootDomain() *RootDomain {
	return &RootDomain{}
}