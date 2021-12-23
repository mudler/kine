package edgevpn

import (
	"context"
	"time"

	"github.com/ipfs/go-log"
	logger "github.com/ipfs/go-log/v2"
	"github.com/k3s-io/kine/pkg/server"
	"github.com/libp2p/go-libp2p"
	"github.com/mudler/edgevpn/pkg/blockchain"
	"github.com/mudler/edgevpn/pkg/edgevpn"
)

// Client implement a shim for the edgevpn ledger connected over the api
type Client struct {
	networkToken string
	namespace    string
	vpn          *edgevpn.EdgeVPN
}

func (c *Client) Start(ctx context.Context) error {
	// First join the network as edgeVPN does.
	llger := logger.Logger("ledger")

	opts := []edgevpn.Option{
		edgevpn.WithDiscoveryInterval(120 * time.Second),
		edgevpn.WithLedgerAnnounceTime(120 * time.Second),
		edgevpn.WithLedgerInterval(120 * time.Second),
		edgevpn.Logger(llger),
		edgevpn.LibP2PLogLevel(log.LevelError),
		edgevpn.WithInterfaceMTU(1200),
		edgevpn.WithPacketMTU(1420),
		edgevpn.FromBase64(true, true, c.networkToken),
	}

	libp2pOpts := []libp2p.Option{

		libp2p.UserAgent("kine"),
		libp2p.EnableAutoRelay(),
		libp2p.AutoNATServiceRateLimit(
			1, 1, 10*time.Second),
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
		libp2p.EnableNATService(),
	}

	opts = append(opts, edgevpn.WithLibp2pOptions(libp2pOpts...))
	opts = append(opts, edgevpn.WithStore(&blockchain.MemoryStore{}))

	e := edgevpn.New(opts...)

	// Join the node to the network, using our ledger
	if err := e.Join(); err != nil {
		return err
	}

	c.vpn = e

	return nil
}

type entry map[string]*server.KeyValue

func (c *Client) Get(ctx context.Context, key string, revision int64) (int64, *server.KeyValue, error) {
	return 0, nil, nil
}

func (c *Client) Create(ctx context.Context, key string, value []byte, lease int64) (int64, error) {
	ledger, err := c.vpn.Ledger()
	if err != nil {
		return 0, err

	}

	kv := &server.KeyValue{
		Key:            key,
		Value:          value,
		Lease:          lease,
		CreateRevision: 0,
		ModRevision:    0,
	}

	e := &entry{"0": kv}

	ledger.Persist(ctx, 5, 15, c.namespace, key, e)

	for {
		d, ok := ledger.GetKey(c.namespace, key)
		if ok {
			break

		}
		d.Unmarshal(e)
	}

	return 0, nil
}

func (c *Client) Delete(ctx context.Context, key string, revision int64) (int64, *server.KeyValue, bool, error)
func (c *Client) List(ctx context.Context, prefix, startKey string, limit, revision int64) (int64, []*server.KeyValue, error)
func (c *Client) Count(ctx context.Context, prefix string) (int64, int64, error)
func (c *Client) Update(ctx context.Context, key string, value []byte, revision, lease int64) (int64, *server.KeyValue, bool, error)
func (c *Client) Watch(ctx context.Context, key string, revision int64) <-chan []*server.Event
func (c *Client) DbSize(ctx context.Context) (int64, error)

func New(ctx context.Context, datasourceName string) (server.Backend, error) {
	// datasourceName is our networktoken
	return &Client{networkToken: datasourceName, namespace: "default"}, nil
}
