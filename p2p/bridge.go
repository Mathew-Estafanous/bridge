package p2p

import (
	"context"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-kad-dht"
)

type Bridge struct {
	host host.Host
	dht *dht.IpfsDHT
}

func (b *Bridge) ID() string {
	return b.host.ID().String()
}

func (b *Bridge) Close() error {
	return b.host.Close()
}

func NewBridge(ctx context.Context) (*Bridge, error) {
	host, err:= libp2p.New()
	if err != nil {
		return nil, err
	}
	kdht, err := newDHT(ctx, host)
	if err != nil {
		return nil, err
	}
	return &Bridge{
		host: host,
		dht: kdht,
	}, nil
}

func newDHT(ctx context.Context, host host.Host) (*dht.IpfsDHT, error) {
	kdht, err := dht.New(ctx, host, dht.Mode(dht.ModeServer))
	if err != nil {
		return nil, err
	}
	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err
	}
	return kdht, nil
}

//func discover(ctx context.Context, host host.Host, dht *dht.IpfsDHT, rendezvous string) {
//	routeDisc := disc.NewRoutingDiscovery(dht)
//	disc.Advertise(ctx, routeDisc, rendezvous)
//
//}

