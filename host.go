package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	mrand "math/rand"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/multiformats/go-multiaddr"
)

func NewHost(ctx context.Context, seed int64, port int) (host.Host, error) {
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
		return nil, err
	}

	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))

	host, err := libp2p.New(ctx,
		libp2p.ListenAddrs(addr),
		libp2p.Identity(priv),
	)
	if err != nil {
		return nil, err
	}

	log.Printf("[%s] Listening on %s\n", host.ID()[0:6].Pretty(), addr)

	return host, nil
}
