package p2p

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"log"
	"time"
)

var BridgeServiceTag = "bridge-service"

type Bridge struct {
	h host.Host
	t *pubsub.Topic
}

func (b *Bridge) HandlePeerFound(info peer.AddrInfo) {
	if info.ID == b.h.ID() {
		return
	}
	log.Printf("Discovered new peer %s", info.ID.Pretty())
	err := b.h.Connect(context.Background(), info)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", info.ID.Pretty(), err)
	}
}

func (b *Bridge) Session() string {
	return b.t.String()
}

func (b *Bridge) Close() error {
	return b.h.Close()
}

func NewBridge(ctx context.Context) (*Bridge, error) {
	host, err:= libp2p.New()
	if err != nil {
		return nil, err
	}

	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, err
	}

	sessionId := uuid.New()
	topic, err := ps.Join(sessionId.String())
	if err != nil {
		return nil, err
	}
	bridge := &Bridge{
		h: host,
		t: topic,
	}
	if err := setupDiscovery(host, bridge); err != nil {
		return nil, err
	}

	go func() {
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		for  {
			select {
			case <-ctx.Done():
				return
			case <- ticker.C:
				topic.Publish(context.Background(), []byte("hello"))
			}
		}
	}()
	return bridge, nil
}

func setupDiscovery(host host.Host, notifee mdns.Notifee) error {
	s := mdns.NewMdnsService(host, BridgeServiceTag, notifee)
	return s.Start()
}

type Client struct {
	h host.Host
	s *pubsub.Subscription
}

func (c *Client) HandlePeerFound(info peer.AddrInfo) {
	if info.ID == c.h.ID() {
		return
	}
	log.Printf("Discovered new peer %s", info.ID.Pretty())
	err := c.h.Connect(context.Background(), info)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", info.ID.Pretty(), err)
	}
}

func (c *Client) Close() error {
	return c.h.Close()
}

func (c *Client) listen() {
	for {
		msg, err := c.s.Next(context.Background())
		if err != nil {
			return
		}
		fmt.Println(string(msg.Data))
	}
}

func NewClient(ctx context.Context, sessionID string) (*Client, error) {
	host, err:= libp2p.New()
	if err != nil {
		return nil, err
	}

	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, err
	}
	topic, err := ps.Join(sessionID)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}
	c := &Client{
		h: host,
		s: sub,
	}
	if err := setupDiscovery(host, c); err != nil {
		return nil, err
	}

	go c.listen()
	return c, nil
}
