// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package message

import (
	"context"
	"errors"

	"github.com/chanmineom/easy-chat/apps/im/api/internal/svc"
	"github.com/chanmineom/easy-chat/apps/im/api/internal/types"
	"github.com/chanmineom/easy-chat/apps/im/rpc/imclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type SyncMessagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// sync messages after a conversation sequence
func NewSyncMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncMessagesLogic {
	return &SyncMessagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SyncMessagesLogic) SyncMessages(req *types.SyncMessagesReq) (resp *types.SyncMessagesResp, err error) {
	// todo: add your logic here and delete this line
	userID, ok := l.ctx.Value("easy-chat-uid").(string)
	if !ok || userID == "" {
		return nil, errors.New("authenticated user id is missing")
	}
	if req.ConversationId == "" {
		return nil, errors.New("conversation_id is required")
	}
	if req.AfterSeq < 0 {
		return nil, errors.New("after_seq must not be negative")
	}
	if req.BeforeSeq < 0 {
		return nil, errors.New(
			"before_seq must not be negative",
		)
	}
	if req.AfterSeq > 0 && req.BeforeSeq > 0 {
		return nil, errors.New(
			"after_seq and before_seq cannot be used together",
		)
	}

	rpcResp, err := l.svcCtx.Im.GetMessages(
		l.ctx,
		&imclient.GetMessagesReq{
			UserId:         userID,
			ConversationId: req.ConversationId,
			AfterSeq:       req.AfterSeq,
			BeforeReq:      req.BeforeSeq,
			Limit:          req.Limit,
		},
	)
	if err != nil {
		return nil, err
	}

	list := make([]*types.SyncMessage, 0, len(rpcResp.Messages))
	for _, message := range rpcResp.Messages {
		list = append(list, &types.SyncMessage{
			MessageId:       message.MessageId,
			ConversationId:  message.ConversationId,
			Sequence:        message.Sequence,
			SenderId:        message.SenderId,
			ClientMessageId: message.ClientMessageId,
			ContentType:     message.ContentType,
			Content:         message.Content,
			CreatedAt:       message.CreatedAt,
		})
	}

	return &types.SyncMessagesResp{
		List:    list,
		HasMore: rpcResp.HasMore,
	}, nil
}
