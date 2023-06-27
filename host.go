package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/multiformats/go-multiaddr"
)

func NewHost(ctx context.Context, seed int64, port int) (host.Host, bool, error) {

	isValidator := false

	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs

	var r io.Reader
	var crypto_code int

	if seed == 0 {
		r = rand.Reader
		crypto_code = crypto.RSA
	} else {
		r = mrand.New(mrand.NewSource(seed))
		crypto_code = crypto.Ed25519
	}

	priv, _, err := crypto.GenerateKeyPairWithReader(crypto_code, 2048, r)
	if err != nil {
		return nil, isValidator, err
	}

	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))

	host, err := libp2p.New(ctx,
		libp2p.ListenAddrs(addr),
		libp2p.Identity(priv),
	)

	if err != nil {
		return nil, isValidator, err
	}

	// Randomly choose if this node is a validator or not
	if mrand.Intn(2) == 0 {
		isValidator = true
	}

	return host, isValidator, nil
}
