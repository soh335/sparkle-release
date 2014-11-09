package main

import (
	"crypto/dsa"
	"crypto/rand"
	"crypto/sha1"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
)

// https://groups.google.com/forum/#!topic/golang-nuts/0WLCbeG1vfY
// http://stackoverflow.com/questions/8693513/go-der-and-handling-big-integers

type dsaPrivateKey struct {
	Version       int
	P, Q, G, Y, X *big.Int
}

type dssSigValue struct {
	R, S *big.Int
}

func sign(f string, key string) (string, error) {
	byt, err := ioutil.ReadFile(key)

	if err != nil {
		return "", err
	}

	block, _ := pem.Decode(byt)

	var rawPriv dsaPrivateKey
	rest, err := asn1.Unmarshal(block.Bytes, &rawPriv)
	if len(rest) != 0 {
		return "", fmt.Errorf("asn1 unmarshal seems to be failed")
	}

	if err != nil {
		return "", err
	}

	priv := &dsa.PrivateKey{
		PublicKey: dsa.PublicKey{
			Parameters: dsa.Parameters{
				P: rawPriv.P,
				Q: rawPriv.Q,
				G: rawPriv.G,
			},
			Y: rawPriv.Y,
		},
		X: rawPriv.X,
	}

	byt, err = ioutil.ReadFile(f)
	if err != nil {
		return "", err
	}

	h := sha1.New()
	if _, err := h.Write(byt); err != nil {
		return "", err
	}
	hash := h.Sum(nil)
	h.Reset()
	if _, err := h.Write(hash); err != nil {
		return "", err
	}

	r, s, err := dsa.Sign(rand.Reader, priv, h.Sum(nil))
	if err != nil {
		return "", err
	}

	sigValue := dssSigValue{r, s}
	byt, err = asn1.Marshal(sigValue)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(byt), nil
}
