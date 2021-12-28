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
	"io"
	"io/ioutil"
	"log"
)

// Bridge is the running instance that can be connected to and
// interacted with.
type Bridge interface {
	io.Closer
	// Session returns the bridge's session id.
	Session() string

	// JoinedPeerListener provides a channel that can be used to listen
	// for a new peer the joins.
	JoinedPeerListener() <- chan Peer

	// Send will stream the data to the peer with the given id.
	Send(id string, data []byte) error
}

// Peer is general information regarding a peer within the system.
type Peer struct {
	Id string
}

type bridge struct {
	session string
	h      host.Host
	joinCh chan Peer
}

func (b *bridge) JoinedPeerListener() <- chan Peer {
	return b.joinCh
}

func (b *bridge) Session() string {
	return b.session
}

func (b *bridge) Close() error {
	close(b.joinCh)
	return b.h.Close()
}

func (b *bridge) Send(id string, data []byte) error {
	peerId, err := peer.Decode(id)
	if err != nil {
		return err
	}

	strm, err := b.h.NewStream(context.Background(), peerId, protocol.ID(b.session))
	if err != nil {
		return err
	}
	if _, err := strm.Write(data); err != nil {
		return err
	}
	return strm.Close()
}

func (b *bridge) HandlePeerFound(info peer.AddrInfo) {
	if info.ID == b.h.ID() {
		return
	}
	log.Printf("Discovered new peer %s", info.ID.Pretty())
	err := b.h.Connect(context.Background(), info)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", info.ID.Pretty(), err)
	}
	b.joinCh <- Peer{info.ID.Pretty()}
}

func NewBridge(testMode bool) (Bridge, error) {
	host, err:= libp2p.New()
	if err != nil {
		return nil, err
	}

	var sessionId string
	if testMode {
		sessionId = "test"
	} else {
		sessionId = uuid.New().String()
	}

	bridge := &bridge{
		session: sessionId,
		h: host,
		joinCh: make(chan Peer, 1),
	}
	
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

	host.SetStreamHandler(protocol.ID(sessionID), c.handleMessage)
	if err := setupDiscovery(host, sessionID, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) handleMessage(strm network.Stream) {
	b, err := ioutil.ReadAll(strm)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(b))
}
