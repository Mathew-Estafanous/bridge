package p2p

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"log"
)

// Peer is general information regarding a peer within the system.
type Peer struct {
	Id string
}

// Bridge is the running instance that can be connected to and
// interacted with.
type Bridge struct {
	session string
	h      host.Host
	joinCh chan Peer
}

func (b *Bridge) JoinedPeerListener() <- chan Peer {
	return b.joinCh
}

func (b *Bridge) Session() string {
	return b.session
}

func (b *Bridge) Close() error {
	close(b.joinCh)
	return b.h.Close()
}

func (b *Bridge) Send(id string, data []byte) error {
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

func (b *Bridge) HandlePeerFound(info peer.AddrInfo) {
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

func NewBridge(testMode bool) (*Bridge, error) {
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

	brg := &Bridge{
		session: sessionId,
		h: host,
		joinCh: make(chan Peer, 1),
	}
	
	if err := setupDiscovery(host, sessionId, brg); err != nil {
		return nil, err
	}
	return brg, nil
}

func setupDiscovery(host host.Host, ns string, notifee mdns.Notifee) error {
	s := mdns.NewMdnsService(host, ns, notifee)
	return s.Start()
}
