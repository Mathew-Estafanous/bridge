package p2p

import (
	"encoding/binary"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

var reg = regexp.MustCompile("(/[^/]*)+$")

type Client struct {
	s string
	h host.Host
}

func (c *Client) HandlePeerFound(_ peer.AddrInfo) {}

func (c *Client) Close() error {
	return c.h.Close()
}

func NewClient(sessionID string) (*Client, error) {
	host, err := libp2p.New()
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
	defer strm.Close()
	b := make([]byte, 5)
	if _, err := strm.Read(b); err != nil {
		log.Println(err)
		return
	}
	pathLn := binary.LittleEndian.Uint32(b)
	pathB := make([]byte, pathLn)
	if _, err := strm.Read(pathB); err != nil {
		log.Println(err)
		return
	}
	path := string(pathB)
	i := strings.LastIndex(path, "/")
	dir := path[:i]
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0777); err != nil {
			log.Println(err)
			return
		}
	}

	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		log.Println(err)
		return
	}

	if _, err := io.Copy(f, strm); err != nil {
		log.Println(err)
		return
	}
}
