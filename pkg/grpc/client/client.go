package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/v2/tss"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	pb "github.com/smiletrl/tss-lib-starter/pkg/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/smiletrl/tss-lib-starter/pkg/constants"
)

// p2p client
type Client interface {
	// with party id
	WithPartyID(pid *tss.PartyID)

	// broadcast message to all nodes
	BroadcastNodes(ctx context.Context, msgType constants.MessageType, msg tss.Message) error

	// send message to one node
	ToNode(ctx context.Context, pid string, msgType constants.MessageType, msg tss.Message) error
}

type client struct {
	// grpc host data, key is party unique id, value is an array with fixed length 2: host + port.
	hosts map[string][2]string

	// it holds all parties grpc client, key is party unique id
	clients    map[string]pb.P2PClient
	clientOnce *sync.Once

	// party unique id
	pid *tss.PartyID
}

func NewClient(hosts map[string][2]string) (Client, error) {
	c := &client{
		hosts:      hosts,
		clientOnce: &sync.Once{},
	}

	return c, nil
}

func (c *client) WithPartyID(pid *tss.PartyID) {
	c.pid = pid
}

// lazy load because grpc server might not be running when client is initialized.
func (c *client) grpc() map[string]pb.P2PClient {
	c.clientOnce.Do(func() {
		tempClients := make(map[string]pb.P2PClient, len(c.hosts))
		for i, host := range c.hosts {
			conn, err := c.newConnectionClient(host[0], host[1])
			if err != nil {
				panic("error new grpc client:" + err.Error())
			}
			tempClients[i] = conn
		}
		c.clients = tempClients
	})
	return c.clients
}

func (c *client) newConnectionClient(host, port string) (client pb.P2PClient, err error) {
	var address = fmt.Sprintf("%s:%s", host, port)

	var kacp = keepalive.ClientParameters{
		Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
		Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
		PermitWithoutStream: true,             // send pings even without active streams
	}

	// only allow maximum 1 second connection.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, address,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(kacp),
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
		)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
		)),
	)
	if err != nil {
		return nil, err
	}
	return pb.NewP2PClient(conn), nil
}

func (c *client) BroadcastNodes(ctx context.Context, msgType constants.MessageType, msg tss.Message) error {
	// broadcast message to all party nodes
	msgID := msg.GetFrom().GetId()
	bz, _, err := msg.WireBytes()
	if err != nil {
		return fmt.Errorf("error getting wire bytes: %w", err)
	}

	for id, g := range c.grpc() {
		// should not send to itself
		if id == msgID {
			continue
		}

		// for signing, if this party is not selected in this round, continue
		if msgType == constants.MessageTypeSigning {
			if _, ok := constants.SelectedParties[id]; !ok {
				continue
			}
		}

		if _, err := g.OnReceiveMessage(ctx, &pb.Message{
			Type:        string(msgType),
			Content:     bz,
			IsBroadcast: true,
			FromPid:     msgID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (c *client) ToNode(ctx context.Context, pid string, msgType constants.MessageType, msg tss.Message) error {
	g, ok := c.grpc()[pid]
	if !ok {
		return fmt.Errorf("unexpected party unique id: %s", pid)
	}

	// for signing, if this party is not selected in this round, continue
	if msgType == constants.MessageTypeSigning {
		if _, ok := constants.SelectedParties[pid]; !ok {
			return fmt.Errorf("unexpected to node request: %s", pid)
		}
	}

	bz, _, err := msg.WireBytes()
	if err != nil {
		return fmt.Errorf("error getting wire bytes: %w", err)
	}
	if _, err := g.OnReceiveMessage(ctx, &pb.Message{
		Type:        string(msgType),
		Content:     bz,
		IsBroadcast: false,
		FromPid:     c.pid.GetId(),
	}); err != nil {
		return fmt.Errorf("party with unique id %s fails receiving message: %w", pid, err)
	}
	return nil
}
