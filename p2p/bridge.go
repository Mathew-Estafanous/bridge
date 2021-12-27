package p2p

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"log"
)

type Bridge struct {
	session uuid.UUID
	h host.Host
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
	return b.session.String()
}

func (b *Bridge) Close() error {
	return b.h.Close()
}

func NewBridge() (*Bridge, error) {
	host, err:= libp2p.New()
	if err != nil {
		return nil, err
	}

	sessionId := uuid.New()
	bridge := &Bridge{
		session: sessionId,
		h: host,
	}
	if err := setupDiscovery(host, sessionId.String(), bridge); err != nil {
		return nil, err
	}
	return bridge, nil
}

func setupDiscovery(host host.Host, ns string, notifee mdns.Notifee) error {
	s := mdns.NewMdnsService(host, ns, notifee)
	return s.Start()
}

type Client struct {
	s string
	h host.Host
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

func NewClient(sessionID string) (*Client, error) {
	host, err:= libp2p.New()
	if err != nil {
		return nil, err
	}
	c := &Client{
		h: host,
		s: sessionID,
	}
	if err := setupDiscovery(host, sessionID, c); err != nil {
		return nil, err
	}
	return c, nil
}
