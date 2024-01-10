package main

import (
	//"context"

	"context"
	"encoding/json"
	"log"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const headerBufSize = 128

var headerSize = 512 //bytes

type Pub struct {
	host     host.Host
	ctx      context.Context
	ps       *pubsub.PubSub
	topic    *pubsub.Topic
	sub      *pubsub.Subscription
	messages chan *HeaderMessage
}

type HeaderMessage struct {
	SenderID string `json:"SenderID"`
	BlockID  int    `json:"BlockID"`
	Header   []byte `json:"Samples"` // List of samples in a response
}

func CreatePubSub(h host.Host, ctx context.Context) (*Pub, error) {
	// Create a new PubSub instance and connect to topic
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Fatal(err)
	}

	topic, err := ps.Join("header-dissemination")
	if err != nil {
		return nil, err
	}

	// and subscribe to it
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	return &Pub{
		host:     h,
		ctx:      ctx,
		ps:       ps,
		topic:    topic,
		sub:      sub,
		messages: make(chan *HeaderMessage, headerBufSize),
	}, nil
}

func (p *Pub) HeaderPublish(blockID int) error {

	m := &HeaderMessage{
		SenderID: p.host.ID().String(),
		BlockID:  blockID,
		Header:   make([]byte, headerSize),
	}
	msgBytes, err := json.Marshal(m)
	if err != nil {
		return err
	}

	log.Printf("\033[31m Block %d Header Publish \033[0m", blockID)
	return p.topic.Publish(p.ctx, msgBytes)
}

func (p *Pub) readLoop() {
	for {
		msg, err := p.sub.Next(p.ctx)
		if err != nil {
			close(p.messages)
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == p.host.ID() {
			continue
		}
		cm := new(HeaderMessage)
		err = json.Unmarshal(msg.Data, cm)
		if err != nil {
			continue
		}
		// send valid messages onto the Messages channel
		p.messages <- cm
	}
}
