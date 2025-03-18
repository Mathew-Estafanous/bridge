package p2p

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"io"
)

type Client struct {
	s        string
	h        host.Host
	streamCh chan io.ReadCloser
}

func (c *Client) HandlePeerFound(_ peer.AddrInfo) {}

func (c *Client) Close() error {
	close(c.streamCh)
	return c.h.Close()
}

func NewClient(sessionID string) (*Client, error) {
	p2pHost, err := libp2p.New()
	if err != nil {
		return nil, err
	}
	c := &Client{
		h:        p2pHost,
		s:        sessionID,
		streamCh: make(chan io.ReadCloser, 1),
	}

	p2pHost.SetStreamHandler(protocol.ID(sessionID), c.handleMessage)
	if err := setupDiscovery(p2pHost, sessionID, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) ListenForStream() <-chan io.ReadCloser {
	return c.streamCh
}

func (c *Client) handleMessage(strm network.Stream) {
	c.streamCh <- strm
}
