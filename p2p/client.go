package p2p

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"io"
	"regexp"
)

var reg = regexp.MustCompile("(/[^/]*)+$")

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
	host, err := libp2p.New()
	if err != nil {
		return nil, err
	}
	c := &Client{
		h:        host,
		s:        sessionID,
		streamCh: make(chan io.ReadCloser, 1),
	}

	host.SetStreamHandler(protocol.ID(sessionID), c.handleMessage)
	if err := setupDiscovery(host, sessionID, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) ListenForStream() <-chan io.ReadCloser {
	return c.streamCh
}

func (c *Client) handleMessage(strm network.Stream) {
	c.streamCh <- strm
	return
}
