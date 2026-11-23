package gatewayrouter

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/session"
	gatewaypb "github.com/chanmineom/easy-chat/apps/im/ws/rpc/gateway"
	"github.com/chanmineom/easy-chat/apps/im/ws/rpc/gatewayclient"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/gatewayport"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

var _ gatewayport.Gateway = (*Router)(nil)

type nodeClient struct {
	raw     zrpc.Client
	gateway gatewayclient.Gateway
}

type Router struct {
	sessions  session.Registry
	etcdHosts []string
	keyPrefix string
	app       string
	token     string

	mu      sync.Mutex
	clients map[string]*nodeClient
	timeout time.Duration
}

func New(sessions session.Registry, etcdHosts []string, keyPrefix string, app string, token string) (*Router, error) {
	if len(etcdHosts) == 0 {
		return nil, fmt.Errorf("etcd hosts are required")
	}

	keyPrefix = strings.TrimSpace(keyPrefix)
	if keyPrefix == "" {
		return nil, fmt.Errorf("ws rpc etcd key prefix is required")
	}

	if app == "" || token == "" {
		return nil, fmt.Errorf("internal rpc app and token are required")
	}

	return &Router{
		sessions:  sessions,
		etcdHosts: etcdHosts,
		keyPrefix: strings.TrimRight(keyPrefix, "/") + "/",
		app:       app,
		token:     token,
		clients:   make(map[string]*nodeClient),
		timeout:   5 * time.Second,
	}, nil
}

func (r *Router) PushToUsers(ctx context.Context, in *gatewaypb.PushToUsersReq) (*gatewaypb.PushToUsersResp, error) {
	if in == nil || in.Message == nil {
		return nil, fmt.Errorf("message is required")
	}

	nodeUsers := make(map[string]map[string]struct{})
	onlineUsers := make(map[string]struct{})

	for _, userID := range in.UserIds {
		if userID == "" {
			continue
		}

		sessions, err := r.sessions.Sessions(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("load sessions for user %s: %w", userID, err)
		}
		if len(sessions) == 0 {
			continue
		}

		// onlineUsers[userID] = struct{}{}

		for _, wsSession := range sessions {
			if wsSession.NodeID == "" {
				logx.WithContext(ctx).Errorf(
					"session has no node id user=%s session=%s",
					userID,
					wsSession.SessionID,
				)
				continue
			}
			// if wsSession.RPCAddr == "" {
			// 	continue
			// }

			onlineUsers[userID] = struct{}{}

			if nodeUsers[wsSession.NodeID] == nil {
				nodeUsers[wsSession.NodeID] = make(map[string]struct{})
			}
			nodeUsers[wsSession.NodeID][userID] = struct{}{}
		}
	}

	var deliveredConnections int32

	// for rpcAddr, users := range nodeUsers {
	for nodeID, users := range nodeUsers {
		client, err := r.clientForNode(nodeID)
		if err != nil {
			return nil, err
		}

		userIDs := make([]string, 0, len(users))
		for userID := range users {
			userIDs = append(userIDs, userID)
		}

		response, err := client.PushToUsers(ctx, &gatewaypb.PushToUsersReq{
			UserIds: userIDs,
			Message: in.Message,
		})
		if err != nil {
			return nil, fmt.Errorf("push to gateway node %s: %w", nodeID, err)
		}

		deliveredConnections += response.DeliveredConnections
	}

	logx.WithContext(ctx).Infof(
		"routed message users=%d online_users=%d nodes=%d delivered_connections=%d",
		len(in.UserIds),
		len(onlineUsers),
		len(nodeUsers),
		deliveredConnections,
	)

	return &gatewaypb.PushToUsersResp{
		OnlineUsers:          int32(len(onlineUsers)),
		DeliveredConnections: deliveredConnections,
	}, nil
}

func (r *Router) PushCommandResult(ctx context.Context, in *gatewaypb.PushCommandResultReq) (*gatewaypb.PushCommandResultResp, error) {
	if in == nil || in.Result == nil || in.UserId == "" {
		return nil, fmt.Errorf("command result and user id are required")
	}

	sessions, err := r.sessions.Sessions(ctx, in.UserId)
	if err != nil {
		return nil, fmt.Errorf("load command result sessions: %w", err)
	}

	nodeIDs := make(map[string]struct{})
	// addresses := make(map[string]struct{})
	for _, wsSession := range sessions {
		if wsSession.NodeID != "" {
			nodeIDs[wsSession.NodeID] = struct{}{}
		}
	}

	for nodeID := range nodeIDs {
		client, err := r.clientForNode(nodeID)
		if err != nil {
			return nil, err
		}

		if _, err := client.PushCommandResult(
			ctx,
			&gatewaypb.PushCommandResultReq{
				UserId: in.UserId,
				Result: in.Result,
			},
		); err != nil {
			return nil, fmt.Errorf(
				"push command result to gateway node %s: %w",
				nodeID,
				err,
			)
		}
	}

	return &gatewaypb.PushCommandResultResp{}, nil
}

// func (r *Router) clientFor(rpcAddr string) (gatewayclient.Gateway, error) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()

// 	if existing := r.clients[rpcAddr]; existing != nil {
// 		return existing.gateway, nil
// 	}

// 	raw, err := zrpc.NewClient(zrpc.RpcClientConf{
// 		Endpoints: []string{rpcAddr},
// 		NonBlock:  true,
// 		Timeout:   int64(r.timeout / time.Millisecond),
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf(
// 			"create gateway client for %s: %w",
// 			rpcAddr,
// 			err,
// 		)
// 	}

// 	entry := &nodeClient{
// 		raw:     raw,
// 		gateway: gatewayclient.NewGateway(raw),
// 	}
// 	r.clients[rpcAddr] = entry

// 	return entry.gateway, nil
// }

func (r *Router) clientForNode(nodeID string) (gatewayclient.Gateway, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing := r.clients[nodeID]; existing != nil {
		return existing.gateway, nil
	}

	key := r.keyPrefix + nodeID

	raw, err := zrpc.NewClient(zrpc.RpcClientConf{
		Etcd: discov.EtcdConf{
			Hosts: r.etcdHosts,
			Key:   key,
		},
		App: r.app,
		Token: r.token,
		NonBlock: true,
		Timeout:  int64(r.timeout / time.Millisecond),
	})
	if err != nil {
		return nil, fmt.Errorf(
			"create gateway client node=%s key=%s: %w",
			nodeID,
			key,
			err,
		)
	}

	entry := &nodeClient{
		raw:     raw,
		gateway: gatewayclient.NewGateway(raw),
	}
	r.clients[nodeID] = entry

	return entry.gateway, nil
}

func (r *Router) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var firstErr error
	// for address, entry := range r.clients {
	// 	if err := entry.raw.Conn().Close(); err != nil && firstErr == nil {
	// 		firstErr = fmt.Errorf(
	// 			"close gateway client %s: %w",
	// 			address,
	// 			err,
	// 		)
	// 	}
	// }
	for nodeID, entry := range r.clients {
		if err := entry.raw.Conn().Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf(
				"close gateway client node=%s: %w",
				nodeID,
				err,
			)
		}
	}
	r.clients = make(map[string]*nodeClient)

	return firstErr
}
