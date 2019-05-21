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

package member

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/insolar/insolar/application/contract/member/signer"
	"github.com/insolar/insolar/application/proxy/deposit"
	"github.com/insolar/insolar/application/proxy/member"
	"github.com/insolar/insolar/application/proxy/nodedomain"
	"github.com/insolar/insolar/application/proxy/rootdomain"
	"github.com/insolar/insolar/application/proxy/wallet"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/logicrunner/goplugin/foundation"
	"golang.org/x/crypto/sha3"
	"math"
)

type Member struct {
	foundation.BaseContract
	Name      string
	EthAddr   string
	PublicKey string
}

func (m *Member) GetName() (string, error) {
	return m.Name, nil
}

func (m *Member) GetEthAddr() (string, error) {
	return m.EthAddr, nil
}

func (m *Member) SetEthAddr(ethAddr string) error {
	m.EthAddr = ethAddr
	return nil
}

var INSATTR_GetPublicKey_API = true

func (m *Member) GetPublicKey() (string, error) {
	return m.PublicKey, nil
}

func New(ethAddr string, key string) (*Member, error) {
	return &Member{
		EthAddr:   ethAddr,
		PublicKey: key,
	}, nil
}

func NewOracleMember(name string, key string) (*Member, error) {
	return &Member{
		Name:      name,
		PublicKey: key,
	}, nil
}

func (m *Member) verifySig(method string, params []byte, seed []byte, sign []byte) error {
	//args, err := insolar.MarshalArgs(m.GetReference(), method, params, seed)
	//if err != nil {
	//	return fmt.Errorf("[ verifySig ] Can't MarshalArgs: %s", err.Error())
	//}

	args, err := json.Marshal(struct {
		Reference string `json:"reference"`
		Method    string `json:"method"`
		Params    string `json:"params"`
		Seed      string `json:"seed"`
	}{
		Reference: m.GetReference().String(),
		Method:    method,
		Params:    string(params),
		Seed:      string(seed),
	})
	if err != nil {
		return fmt.Errorf("[ verifySig ] Can't json Marshal: %s", err.Error())
	}
	key, err := m.GetPublicKey()
	if err != nil {
		return fmt.Errorf("[ verifySig ]: %s", err.Error())
	}

	publicKey, err := foundation.ImportPublicKey(key)
	if err != nil {
		return fmt.Errorf("[ verifySig ] Invalid public key")
	}

	verified, err := foundation.Verify(args, sign, publicKey)
	if err != nil {
		return fmt.Errorf("[ verifySig ] Cant verify: %s", err.Error())
	}
	if !verified {
		return fmt.Errorf("[ verifySig ] Incorrect signature1")
	}
	return nil
}

var INSATTR_Call_API = true

// Call method for authorized calls
func (m *Member) Call(rootDomainRef insolar.Reference, method string, params []byte, seed []byte, sign []byte) (interface{}, error) {

	switch method {
	case "CreateMember":
		return m.createMemberCall(rootDomainRef, params)
	case "CreateOracleMember":
		return m.createOracleMemberCall(rootDomainRef, params)
	}

	if err := m.verifySig(method, params, seed, sign); err != nil {
		return nil, fmt.Errorf("[ Call ]: %s", err.Error())
	}

	switch method {
	case "GetMyBalance":
		return m.getMyBalanceCall()
	case "GetBalance":
		return m.getBalanceCall(params)
	case "Transfer":
		return m.transferCall(params)
	case "DumpUserInfo":
		return m.dumpUserInfoCall(rootDomainRef, params)
	case "DumpAllUsers":
		return m.dumpAllUsersCall(rootDomainRef)
	case "RegisterNode":
		return m.registerNodeCall(rootDomainRef, params)
	case "GetNodeRef":
		return m.getNodeRefCall(rootDomainRef, params)
	case "Migration":
		return m.migration(rootDomainRef, params)
	}
	return nil, &foundation.Error{S: "Unknown method"}
}

func verifyKey(key string) (bool, error) {
	if key == "" {
		return false, nil
	} else {
		return true, nil
	}
}

func (m *Member) createMemberCall(rdRef insolar.Reference, params []byte) (interface{}, error) {
	rootDomain := rootdomain.GetObject(rdRef)
	var name string
	var key string
	if err := signer.UnmarshalParams(params, &name, &key); err != nil {
		return nil, fmt.Errorf("[ createMemberCall ]: %s", err.Error())
	}

	//valid, err := verifyKey(key)
	//if err != nil {
	//	return nil, fmt.Errorf("[ createMemberCall ] Can't verify key: %s", err.Error())
	//}
	//if !valid {
	//	return nil, fmt.Errorf("[ createMemberCall ] Key is not valid: %s", err.Error())
	//}

	return rootDomain.CreateMember(name, key)
}

func (m *Member) createOracleMemberCall(rdRef insolar.Reference, params []byte) (interface{}, error) {
	var ethAddr string
	var key string
	if err := signer.UnmarshalParams(params, &ethAddr, &key); err != nil {
		return nil, fmt.Errorf("[ createOracleMemberCall ]: %s", err.Error())
	}

	valid, err := verifyKey(key)
	if err != nil {
		return nil, fmt.Errorf("[ createOracleMemberCall ] Can't verify key: %s", err.Error())
	}
	if !valid {
		return nil, fmt.Errorf("[ createOracleMemberCall ] Key is not valid: %s", err.Error())
	}

	memberHolder := member.New(ethAddr, key)
	new, err := memberHolder.AsChild(rdRef)
	if err != nil {
		return nil, fmt.Errorf("[ createOracleMemberCall ] Can't save as child: %s", err.Error())
	}

	wHolder := wallet.New(1000 * 1000 * 1000)
	_, err = wHolder.AsDelegate(new.GetReference())
	if err != nil {
		return nil, fmt.Errorf("[ createOracleMemberCall ] Can't save as delegate: %s", err.Error())
	}

	return m.GetReference().String(), nil
}

func (m *Member) getMyBalanceCall() (interface{}, error) {
	w, err := wallet.GetImplementationFrom(m.GetReference())
	if err != nil {
		return 0, fmt.Errorf("[ getMyBalanceCall ]: %s", err.Error())
	}

	return w.GetBalance()
}

func (m *Member) getBalanceCall(params []byte) (interface{}, error) {
	var member string
	if err := signer.UnmarshalParams(params, &member); err != nil {
		return nil, fmt.Errorf("[ getBalanceCall ] : %s", err.Error())
	}
	memberRef, err := insolar.NewReferenceFromBase58(member)
	if err != nil {
		return nil, fmt.Errorf("[ getBalanceCall ] : %s", err.Error())
	}
	w, err := wallet.GetImplementationFrom(*memberRef)
	if err != nil {
		return nil, fmt.Errorf("[ getBalanceCall ] : %s", err.Error())
	}

	return w.GetBalance()
}

func parseAmount(inAmount interface{}) (amount uint, err error) {
	switch a := inAmount.(type) {
	case uint:
		amount = a
	case uint64:
		if a > math.MaxUint32 {
			return 0, errors.New("Transfer ammount bigger than integer")
		}
		amount = uint(a)
	case float32:
		if a > math.MaxUint32 {
			return 0, errors.New("Transfer ammount bigger than integer")
		}
		amount = uint(a)
	case float64:
		if a > math.MaxUint32 {
			return 0, errors.New("Transfer ammount bigger than integer")
		}
		amount = uint(a)
	default:
		return 0, fmt.Errorf("Wrong type for amount %t", inAmount)
	}

	return amount, nil
}

func (m *Member) transferCall(params []byte) (interface{}, error) {
	var amount uint
	var toStr string
	var inAmount interface{}
	if err := signer.UnmarshalParams(params, &inAmount, &toStr); err != nil {
		return nil, fmt.Errorf("[ transferCall ] Can't unmarshal params: %s", err.Error())
	}

	amount, err := parseAmount(inAmount)
	if err != nil {
		return nil, fmt.Errorf("[ transferCall ] Failed to parse amount: %s", err.Error())
	}

	to, err := insolar.NewReferenceFromBase58(toStr)
	if err != nil {
		return nil, fmt.Errorf("[ transferCall ] Failed to parse 'to' param: %s", err.Error())
	}
	if m.GetReference() == *to {
		return nil, fmt.Errorf("[ transferCall ] Recipient must be different from the sender")
	}
	w, err := wallet.GetImplementationFrom(m.GetReference())
	if err != nil {
		return nil, fmt.Errorf("[ transferCall ] Can't get implementation: %s", err.Error())
	}

	return nil, w.Transfer(amount, to)
}

func (m *Member) registerNodeCall(ref insolar.Reference, params []byte) (interface{}, error) {
	var publicKey string
	var role string
	if err := signer.UnmarshalParams(params, &publicKey, &role); err != nil {
		return nil, fmt.Errorf("[ registerNodeCall ] Can't unmarshal params: %s", err.Error())
	}

	rootDomain := rootdomain.GetObject(ref)
	nodeDomainRef, err := rootDomain.GetNodeDomainRef()
	if err != nil {
		return nil, fmt.Errorf("[ registerNodeCall ] %s", err.Error())
	}

	nd := nodedomain.GetObject(nodeDomainRef)
	cert, err := nd.RegisterNode(publicKey, role)
	if err != nil {
		return nil, fmt.Errorf("[ registerNodeCall ] Problems with RegisterNode: %s", err.Error())
	}

	return string(cert), nil
}

func (m *Member) getNodeRefCall(ref insolar.Reference, params []byte) (interface{}, error) {
	var publicKey string
	if err := signer.UnmarshalParams(params, &publicKey); err != nil {
		return nil, fmt.Errorf("[ getNodeRefCall ] Can't unmarshal params: %s", err.Error())
	}

	rootDomain := rootdomain.GetObject(ref)
	nodeDomainRef, err := rootDomain.GetNodeDomainRef()
	if err != nil {
		return nil, fmt.Errorf("[ getNodeRefCall ] Can't get nodeDmainRef: %s", err.Error())
	}

	nd := nodedomain.GetObject(nodeDomainRef)
	nodeRef, err := nd.GetNodeRefByPK(publicKey)
	if err != nil {
		return nil, fmt.Errorf("[ getNodeRefCall ] NetworkNode not found: %s", err.Error())
	}

	return nodeRef, nil
}

func (m *Member) FindDeposit(txHash string, amount uint) (bool, deposit.Deposit, error) {
	iterator, err := m.NewChildrenTypedIterator(deposit.GetPrototype())
	if err != nil {
		return false, deposit.Deposit{}, fmt.Errorf("[ findDeposit ] Can't get children: %s", err.Error())
	}

	for iterator.HasNext() {
		cref, err := iterator.Next()
		if err != nil {
			return false, deposit.Deposit{}, fmt.Errorf("[ findDeposit ] Can't get next child: %s", err.Error())
		}

		if !cref.IsEmpty() {
			d := deposit.GetObject(cref)
			th, err := d.GetTxHash()
			if err != nil {
				return false, deposit.Deposit{}, fmt.Errorf("[ findDeposit ] Can't get tx hash: %s", err.Error())
			}
			a, err := d.GetAmount()
			if err != nil {
				return false, deposit.Deposit{}, fmt.Errorf("[ findDeposit ] Can't get amount: %s", err.Error())
			}

			if txHash == th {
				if amount == a {
					return true, *d, nil
				}
			}
		}
	}

	return false, deposit.Deposit{}, nil
}

func (mdMember *Member) migration(rdRef insolar.Reference, params []byte) (string, error) {
	if mdMember.Name == "" {
		return "", fmt.Errorf("[ migration ] Only oracles can call migration")
	}

	var txHash, ethAddr, inInsAddr string
	var inAmount interface{}
	if err := signer.UnmarshalParams(params, &txHash, &ethAddr, &inAmount, &inInsAddr); err != nil {
		return "", fmt.Errorf("[ migration ] Can't unmarshal params: %s", err.Error())
	}

	amount, err := parseAmount(inAmount)
	if err != nil {
		return "", fmt.Errorf("[ migration ] Failed to parse amount: %s", err.Error())
	}

	getInsAddress := func() (insolar.Reference, error) {
		var insAddr insolar.Reference
		if inInsAddr == "" {
			memberHolder := member.New(ethAddr, "")
			m, err := memberHolder.AsChild(rdRef)
			if err != nil {
				return [64]byte{}, fmt.Errorf("[ migration ] Can't save as child: %s", err.Error())
			}
			insAddr = m.GetReference()
		} else {
			pInsAddr, err := insolar.NewReferenceFromBase58(inInsAddr)
			if err != nil {
				return [64]byte{}, fmt.Errorf("[ migration ] Failed to parse 'inInsAddr' param: %s", err.Error())
			}
			insAddr = *pInsAddr

		}

		return insAddr, nil
	}
	insAddr, err := getInsAddress()
	if err != nil {
		return "", fmt.Errorf("[ migration ] Can't get insolar address: %s", err.Error())
	}

	insMember := member.GetObject(insAddr)

	validateInsMember := func() error {
		insEthAddr, err := insMember.GetEthAddr()
		if err != nil {
			return fmt.Errorf("[ validateInsMember ] Failed to get ethAddr")
		}
		if insEthAddr != "" {
			if ethAddr != insEthAddr {
				return fmt.Errorf("[ validateInsMember ] Not equal ethereum Addr. ethAddr: " + ethAddr + ". insEthAddr: " + insEthAddr)
			}
		} else {
			err := insMember.SetEthAddr(ethAddr)
			if err != nil {
				return fmt.Errorf("[ validateInsMember ] Failed to set ethAddr")
			}
		}

		return nil
	}
	err = validateInsMember()
	if err != nil {
		return "", fmt.Errorf("[ migration ] Insolar member validation failed: %s", err.Error())
	}

	rd := rootdomain.GetObject(rdRef)
	oracleMembers, err := rd.GetOracleMembers()
	if err != nil {
		return "", fmt.Errorf("[ migration ] Can't get oracles map: %s", err.Error())
	}

	found, txDeposit, err := insMember.FindDeposit(txHash, amount)
	if err != nil {
		return "", fmt.Errorf("[ migration ] Can't get deposit: %s", err.Error())
	}
	if !found {
		oracleConfirms := map[string]bool{}
		for name, _ := range oracleMembers {
			oracleConfirms[name] = false
		}
		dHolder := deposit.New(oracleConfirms, txHash, amount)
		txDepositP, err := dHolder.AsDelegate(insAddr)
		if err != nil {
			return "", fmt.Errorf("[ migration ] Can't save as delegate: %s", err.Error())
		}
		txDeposit = *txDepositP
	}

	if _, ok := oracleMembers[mdMember.Name]; !ok {
		return "", fmt.Errorf("[ getOracleConfirms ] This oracle is not in the list")
	}
	allConfirmed, err := txDeposit.Confirm(mdMember.Name, txHash, amount)
	if err != nil {
		return "", fmt.Errorf("[ migration ] Confirmed failed: %s", err.Error())
	}

	if allConfirmed {
		w, err := wallet.GetImplementationFrom(insAddr)
		if err != nil {
			wHolder := wallet.New(0)
			w, err = wHolder.AsDelegate(insAddr)
			if err != nil {
				return "", fmt.Errorf("[ migration ] Can't save as delegate: %s", err.Error())
			}
		}

		getMdWallet := func() (*wallet.Wallet, error) {
			mdWalletRef, err := rd.GetMDWalletRef()
			if err != nil {
				return nil, fmt.Errorf("[ migration ] Can't get md wallet ref: %s", err.Error())
			}
			mdWallet := wallet.GetObject(*mdWalletRef)

			return mdWallet, nil
		}
		mdWallet, err := getMdWallet()
		if err != nil {
			return "", fmt.Errorf("[ migration ] Can't get mdWallet: %s", err.Error())
		}

		err = mdWallet.Transfer(amount, &w.Reference)
		if err != nil {
			return "", fmt.Errorf("[ migration ] Can't transfer: %s", err.Error())
		}

	}

	return insAddr.String(), nil
}

//////////////////

func (m *Member) dumpUserInfoCall(rdRef insolar.Reference, params []byte) (interface{}, error) {
	var userRefIn string
	if err := signer.UnmarshalParams(params, &userRefIn); err != nil {
		return nil, fmt.Errorf("[ dumpUserInfoCall ] Can't unmarshal params: %s", err.Error())
	}
	userRef, err := insolar.NewReferenceFromBase58(userRefIn)
	if err != nil {
		return nil, fmt.Errorf("[ migration ] Failed to parse 'inInsAddr' param: %s", err.Error())
	}

	rootDomain := rootdomain.GetObject(rdRef)
	rootMember, err := rootDomain.GetRootMemberRef()
	if err != nil {
		return nil, fmt.Errorf("[ DumpUserInfo ] Can't get root member: %s", err.Error())
	}
	if *userRef != m.GetReference() && m.GetReference() != *rootMember {
		return nil, fmt.Errorf("[ DumpUserInfo ] You can dump only yourself")
	}

	return m.DumpUserInfo(rdRef, *userRef)
}

func (m *Member) dumpAllUsersCall(rdRef insolar.Reference) (interface{}, error) {
	rootDomain := rootdomain.GetObject(rdRef)
	rootMember, err := rootDomain.GetRootMemberRef()
	if err != nil {
		return nil, fmt.Errorf("[ DumpUserInfo ] Can't get root member: %s", err.Error())
	}
	if m.GetReference() != *rootMember {
		return nil, fmt.Errorf("[ DumpUserInfo ] You can dump only yourself")
	}

	return m.DumpAllUsers(rdRef)
}

func (rootMember *Member) getUserInfoMap(m *member.Member) (map[string]interface{}, error) {
	w, err := wallet.GetImplementationFrom(m.GetReference())
	if err != nil {
		return nil, fmt.Errorf("[ getUserInfoMap ] Can't get implementation: %s", err.Error())
	}

	name, err := m.GetName()
	if err != nil {
		return nil, fmt.Errorf("[ getUserInfoMap ] Can't get name: %s", err.Error())
	}

	ethAddr, err := m.GetEthAddr()
	if err != nil {
		return nil, fmt.Errorf("[ getUserInfoMap ] Can't get name: %s", err.Error())
	}

	balance, err := w.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("[ getUserInfoMap ] Can't get total balance: %s", err.Error())
	}
	return map[string]interface{}{
		"name":    name,
		"ethAddr": ethAddr,
		"balance": balance,
	}, nil
}

// DumpUserInfo processes dump user info request
func (m *Member) DumpUserInfo(rdRef insolar.Reference, userRef insolar.Reference) ([]byte, error) {

	user := member.GetObject(userRef)
	res, err := m.getUserInfoMap(user)
	if err != nil {
		return nil, fmt.Errorf("[ DumpUserInfo ] Problem with making request: %s", err.Error())
	}

	return json.Marshal(res)
}

// DumpAllUsers processes dump all users request
func (rootMember *Member) DumpAllUsers(rdRef insolar.Reference) ([]byte, error) {

	res := []map[string]interface{}{}

	rootDomain := rootdomain.GetObject(rdRef)
	iterator, err := rootDomain.DumpAllUsers()
	if err != nil {
		return nil, fmt.Errorf("[ DumpAllUsers ] Can't get children: %s", err.Error())
	}

	for iterator.HasNext() {
		cref, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("[ DumpAllUsers ] Can't get next child: %s", err.Error())
		}

		if !cref.IsEmpty() {
			m := member.GetObject(cref)
			userInfo, err := rootMember.getUserInfoMap(m)
			if err != nil {
				return nil, fmt.Errorf("[ DumpAllUsers ] Problem with making request: %s", err.Error())
			}
			res = append(res, userInfo)
		}
	}
	resJSON, _ := json.Marshal(res)
	return resJSON, nil
}

//////

func decodeHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	return b
}

func hash(msg string) string {

	hash := sha3.NewLegacyKeccak256()

	var buf []byte
	hash.Write(decodeHex(msg))
	buf = hash.Sum(nil)

	return hex.EncodeToString(buf)
}
