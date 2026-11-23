// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package message

import (
	"context"
	"errors"
	"strings"

	"github.com/chanmineom/easy-chat/apps/im/api/internal/svc"
	"github.com/chanmineom/easy-chat/apps/im/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type IssueWSTicketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// issue one-time websocket ticket
func NewIssueWSTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IssueWSTicketLogic {
	return &IssueWSTicketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *IssueWSTicketLogic) IssueWSTicket(req *types.IssueWSTicketReq) (resp *types.IssueWSTicketResp, err error) {
	userID, ok := l.ctx.Value("easy-chat-uid").(string)
	if !ok || userID == "" {
		return nil, errors.New("authenticated user id is missing")
	}

	deviceID := strings.TrimSpace(req.DeviceId)
	if deviceID == "" || len(deviceID) > 128 {
		return nil, errors.New("valid device_id is required")
	}

	issued, err := l.svcCtx.Tickets.Issue(
		l.ctx,
		userID,
		deviceID,
	)
	if err != nil {
		l.Errorf("issue websocket ticket: %v", err)
		return nil, errors.New("could not issue websocket ticket")
	}

	return &types.IssueWSTicketResp{
		Ticket:    issued.Ticket,
		ExpiresAt: issued.ExpiresAt.UnixMilli(),
	}, nil
}
