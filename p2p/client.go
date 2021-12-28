package p2p

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"io/ioutil"
	"log"
)

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