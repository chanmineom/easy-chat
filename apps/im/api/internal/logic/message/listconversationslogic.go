// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package message

import (
	"context"
	"errors"

	"github.com/chanmineom/easy-chat/apps/im/api/internal/svc"
	"github.com/chanmineom/easy-chat/apps/im/api/internal/types"
	"github.com/chanmineom/easy-chat/apps/im/model"
	"github.com/chanmineom/easy-chat/apps/im/rpc/imclient"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"github.com/chanmineom/easy-chat/apps/social/rpc/socialclient"
	"github.com/chanmineom/easy-chat/apps/user/rpc/userclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// list user conversations
func NewListConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListConversationsLogic {
	return &ListConversationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListConversationsLogic) ListConversations(req *types.ListConversationsReq) (resp *types.ListConversationsResp, err error) {
	userID, ok := l.ctx.Value("easy-chat-uid").(string)
	if !ok || userID == "" {
		return nil, errors.New("authenticated user id is missing")
	}

	rpcResp, err := l.svcCtx.Im.ListConversations(
		l.ctx,
		&imclient.ListConversationsReq{
			UserId:               userID,
			Limit:                req.Limit,
			BeforeActivityAt:     req.BeforeActivityAt,
			BeforeConversationId: req.BeforeConversationId,
		},
	)
	if err != nil {
		return nil, err
	}

	friendByConversation := l.loadFriendConversationMap(userID)
	groupByID := l.loadGroupMap(userID)
	profiles := make(map[string]*userclient.UserEntity)

	list := make([]*types.ConversationSummary, 0, len(rpcResp.Conversations))

	for _, conversation := range rpcResp.Conversations {
		item := &types.ConversationSummary{
			ConversationId: conversation.ConversationId,
			ChatType:       conversation.ChatType,
			LastSeq:        conversation.LastSeq,
			ReadSeq:        conversation.ReadSeq,
			DeliveredSeq:   conversation.DeliveredSeq,
			UnreadCount:    conversation.UnreadCount,
			LastActivityAt: conversation.LastActivityAt,
		}

		// var lastMessage *types.SyncMessage
		if conversation.LastMessage != nil {
			item.LastMessage = &types.SyncMessage{
				MessageId:       conversation.LastMessage.MessageId,
				ConversationId:  conversation.LastMessage.ConversationId,
				Sequence:        conversation.LastMessage.Sequence,
				SenderId:        conversation.LastMessage.SenderId,
				ClientMessageId: conversation.LastMessage.ClientMessageId,
				ContentType:     conversation.LastMessage.ContentType,
				Content:         conversation.LastMessage.Content,
				CreatedAt:       conversation.LastMessage.CreatedAt,
			}

			senderID := conversation.LastMessage.SenderId
			senderProfile, exists := profiles[senderID]

			if !exists && senderID != "" {
				response, err := l.svcCtx.User.GetUserInfo(
					l.ctx,
					&userclient.GetUserInfoReq{
						UserId: senderID,
					},
				)
				if err == nil && response.User != nil {
					senderProfile = response.User
					profiles[senderID] = senderProfile
				} else if err != nil {
					l.Errorf(
						"load last message sender user=%s: %v",
						senderID,
						err,
					)
				}
			}

			if senderProfile != nil {
				item.LastSenderName = senderProfile.Nickname
			}
		}

		switch conversation.ChatType {
		case int32(model.ChatTypeSingle):
			peerID := friendByConversation[conversation.ConversationId]
			item.PeerUserId = peerID
			item.Title = "Direct Chat"

			if peerID == "" {
				break
			}

			profile, exists := profiles[peerID]
			if !exists {
				response, err := l.svcCtx.User.GetUserInfo(
					l.ctx,
					&userclient.GetUserInfoReq{
						UserId: peerID,
					},
				)
				if err == nil && response.User != nil {
					profile = response.User
					profiles[peerID] = profile
				} else if err != nil {
					l.Errorf(
						"load conversation peer user=%s: %v",
						peerID,
						err,
					)
				}
			}

			if profile != nil {
				item.Title = profile.Nickname
				item.AvatarUrl = profile.AvatarUrl
			} else {
				item.Title = peerID
			}

		case int32(model.ChatTypeGroup):
			item.Title = "Group Chat"

			if group, exists := groupByID[conversation.ConversationId]; exists {
				item.Title = group.Name
				item.AvatarUrl = group.AvatarUrl
			}
		}

		list = append(list, item)
	}

	return &types.ListConversationsResp{
		List:    list,
		HasMore: rpcResp.HasMore,
	}, nil
}

func (l *ListConversationsLogic) loadFriendConversationMap(userID string) map[string]string {
	result := make(map[string]string)

	response, err := l.svcCtx.Social.ListFriends(
		l.ctx,
		&social.ListFriendsReq{
			UserId: userID,
		},
	)
	if err != nil {
		l.Errorf(
			"load friends for conversations: %v",
			err,
		)
		return result
	}

	for _, friendID := range response.FriendIds {
		conversationID, _, err := model.DirectConversationID(
			userID,
			friendID,
		)
		if err != nil {
			continue
		}

		result[conversationID] = friendID
	}

	return result
}

func (l *ListConversationsLogic) loadGroupMap(userID string) map[string]*socialclient.GroupSummary {
	result := make(map[string]*socialclient.GroupSummary)

	response, err := l.svcCtx.Social.ListGroups(
		l.ctx,
		&socialclient.ListGroupsReq{
			UserId: userID,
		},
	)
	if err != nil {
		l.Errorf(
			"load groups for conversations: %v",
			err,
		)
		return result
	}

	for _, group := range response.Groups {
		result[group.GroupId] = group
	}

	return result
}
