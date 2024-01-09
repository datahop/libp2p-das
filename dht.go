package main

import (
	"context"
	"log"

	dht "github.com/libp2p/go-libp2p-kad-dht"
   "github.com/libp2p/go-libp2p/core/host"
)

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

var testPrefix = dht.ProtocolPrefix("/das")

func NewDHT(ctx context.Context, host host.Host, nodeType string) (*dht.IpfsDHT, error) {
	var options []dht.Option

	if nodeType == "nonvalidator" {
		options = append(options, dht.Mode(dht.ModeClient))
	} else {
		options = append(options, dht.Mode(dht.ModeServer))
	}

	options = append(options, dht.NamespacedValidator("das", blankValidator{}))
	options = append(options, testPrefix)

	kdht, err := dht.New(ctx, host, options...)
	if err != nil {
		log.Printf("dht.New() failed")
		return nil, err
	}

	if err = kdht.Bootstrap(ctx); err != nil {
		log.Printf("Error bootstrapping")
		return nil, err
	}

	return kdht, nil
}
