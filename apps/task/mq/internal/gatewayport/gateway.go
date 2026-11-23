package gatewayport

import (
	"context"

	gatewaypb "github.com/chanmineom/easy-chat/apps/im/ws/rpc/gateway"
)

type Gateway interface {
	PushToUsers(ctx context.Context, in *gatewaypb.PushToUsersReq) (*gatewaypb.PushToUsersResp, error)
	PushCommandResult(ctx context.Context, in *gatewaypb.PushCommandResultReq) (*gatewaypb.PushCommandResultResp, error)
}
