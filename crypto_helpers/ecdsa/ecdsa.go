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

package ecdsa

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"math/big"

	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

// MakeHash makes hash from seed
func MakeHash(seed []byte) [32]byte {
	return sha3.Sum256(seed)
}

// GetCurve gets default curve
func GetCurve() elliptic.Curve {
	return elliptic.P256()
}

// Helper-function for exporting ecdsa.PrivateKey to PEM string
func ExportPrivateKey(privateKey *ecdsa.PrivateKey) (string, error) {
	x509Encoded, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", errors.Wrap(err, "[ ExportPrivateKey ]")
	}
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})
	return string(pemEncoded), nil
}

// Helper-function for exporting ecdsa.PublicKey from PEM string
func ExportPublicKey(publicKey *ecdsa.PublicKey) (string, error) {
	x509EncodedPub, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", errors.Wrap(err, "[ ExportPublicKey ]")
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	return string(pemEncodedPub), nil
}

// Helper-function for importing ecdsa.PrivateKey from PEM string
func ImportPrivateKey(pemEncoded string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemEncoded))
	if block == nil {
		return nil, errors.New("[ ImportPrivateKey ] Problems with parsing")
	}
	x509Encoded := block.Bytes
	privateKey, err := x509.ParseECPrivateKey(x509Encoded)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// Helper-function for importing ecdsa.PublicKey from PEM string
func ImportPublicKey(pemPubEncoded string) (*ecdsa.PublicKey, error) {
	blockPub, _ := pem.Decode([]byte(pemPubEncoded))
	if blockPub == nil {
		return nil, errors.New("[ ImportPublicKey ] Problems with parsing")
	}
	x509EncodedPub := blockPub.Bytes
	genericPublicKey, err := x509.ParsePKIXPublicKey(x509EncodedPub)
	if err != nil {
		return nil, err
	}
	publicKey := genericPublicKey.(*ecdsa.PublicKey)
	return publicKey, nil
}

// EcdsaPair represents two ints for ecdsa
type EcdsaPair struct {
	First  *big.Int
	Second *big.Int
}

// Sign signs given seed
func Sign(seed []byte, key *ecdsa.PrivateKey) ([]byte, error) {

	hash := MakeHash(seed)

	r, s, err := ecdsa.Sign(rand.Reader, key, hash[:])

	if err != nil {
		return nil, errors.Wrap(err, "[ Sign ]")
	}

	data, err := asn1.Marshal(EcdsaPair{First: r, Second: s})
	if err != nil {
		return nil, errors.Wrap(err, "[ Sign ]")
	}

	return data, nil
}

// Verifies signature
func Verify(seed []byte, signatureRaw []byte, pubKey string) (bool, error) {
	var ecdsaPair EcdsaPair
	rest, err := asn1.Unmarshal(signatureRaw, &ecdsaPair)
	if err != nil {
		return false, errors.Wrap(err, "[ Verify ]")
	}
	if len(rest) != 0 {
		return false, errors.New("[ Verify ] len of  rest must be 0")
	}

	savedKey, err := ImportPublicKey(pubKey)
	if err != nil {
		return false, errors.Wrap(err, "[ Verify ]")
	}

	hash := MakeHash(seed)

	return ecdsa.Verify(savedKey, hash[:], ecdsaPair.First, ecdsaPair.Second), nil
}

func GeneratePrivateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(GetCurve(), rand.Reader)
}
