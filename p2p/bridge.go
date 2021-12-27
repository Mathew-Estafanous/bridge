package p2p

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"io/ioutil"
	"log"
)

type Bridge struct {
	session string
	h      host.Host
	joinCh chan Peer
}

type Peer struct {
	Id peer.ID
}

func (b *Bridge) WaitForJoinedPeer() <- chan Peer {
	return b.joinCh
}

func (b *Bridge) Session() string {
	return b.session
}

func (b *Bridge) Close() error {
	return b.h.Close()
}

func (b *Bridge) Send(id peer.ID, msg []byte) error {
	strm, err := b.h.NewStream(context.Background(), id, protocol.ID(b.session))
	if err != nil {
		return err
	}
	if _, err := strm.Write(msg); err != nil {
		return err
	}
	return nil
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
	b.joinCh <- Peer{info.ID}
}

func NewBridge(mode string) (*Bridge, error) {
	host, err:= libp2p.New()
	log.Println(host.Addrs())
	if err != nil {
		return nil, err
	}

	var sessionId string
	if mode == "t" {
		sessionId = "test"
	} else {
		sessionId = uuid.New().String()
	}

	bridge := &Bridge{
		session: sessionId,
		h: host,
		joinCh: make(chan Peer, 1),
	}

	host.SetStreamHandler(protocol.ID(sessionId), handleMessage)
	if err := setupDiscovery(host, sessionId, bridge); err != nil {
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

	host.SetStreamHandler(protocol.ID(sessionID), handleMessage)
	if err := setupDiscovery(host, sessionID, c); err != nil {
		return nil, err
	}
	return c, nil
}

func handleMessage(strm network.Stream) {
	b, err := ioutil.ReadAll(strm)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(b))
}
