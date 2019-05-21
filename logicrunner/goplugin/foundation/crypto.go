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

package foundation

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/platformpolicy"
	"github.com/insolar/insolar/platformpolicy/keys"
)

// TODO: this file should be removed

var platformCryptographyScheme = platformpolicy.NewSecpPlatformCryptographyScheme()
var keyProcessor = platformpolicy.NewKeyProcessor()

// Sign signs given seed.
func Sign(data []byte, key keys.PrivateKey) ([]byte, error) {
	signature, err := platformCryptographyScheme.Signer(key).Sign(data)
	if err != nil {
		return nil, err
	}
	return signature.Bytes(), nil
}

// Verify verifies signature.
func Verify(data []byte, signatureRaw []byte, publicKey keys.PublicKey) (bool, error) {
	return platformCryptographyScheme.Verifier(publicKey).Verify("DER", insolar.SignatureFromBytes(signatureRaw), data)
}

func GeneratePrivateKey() (keys.PrivateKey, error) {
	return keyProcessor.GeneratePrivateKey()
}

func ImportPublicKey(publicKey string) (keys.PublicKey, error) {
	return keyProcessor.ImportPublicKeyPEM([]byte(publicKey))
}

func ExportPublicKey(publicKey keys.PublicKey) (string, error) {
	key, err := keyProcessor.ExportPublicKeyPEM(publicKey)
	return string(key), err
}

func ExtractPublicKey(privateKey keys.PrivateKey) keys.PublicKey {
	return keyProcessor.ExtractPublicKey(privateKey)
}
