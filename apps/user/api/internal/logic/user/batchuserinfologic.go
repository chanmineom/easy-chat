// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/chanmineom/easy-chat/apps/user/api/internal/svc"
	"github.com/chanmineom/easy-chat/apps/user/api/internal/types"
	"github.com/chanmineom/easy-chat/apps/user/rpc/userclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxBatchUserIDs = 100

type BatchUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// batch get user info
func NewBatchUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchUserInfoLogic {
	return &BatchUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BatchUserInfoLogic) BatchUserInfo(req *types.BatchUserInfoReq) (resp *types.BatchUserInfoResp, err error) {
	ids, err := parseUserIDs(req.UserIds)
	if err != nil {
		return nil, err
	}

	list := make([]types.UserInfo, 0, len(ids))

	for _, userID := range ids {
		response, err := l.svcCtx.User.GetUserInfo(
			l.ctx,
			&userclient.GetUserInfoReq{
				UserId: userID,
			},
		)
		if status.Code(err) == codes.NotFound {
			continue
		}
		if err != nil {
			return nil, err
		}

		list = append(list, types.UserInfo{
			Id:        response.User.Id,
			Phone:     response.User.Phone,
			Nickname:  response.User.Nickname,
			AvatarUrl: response.User.AvatarUrl,
			Status:    response.User.Status,
		})
	}

	return &types.BatchUserInfoResp{
		List: list,
	}, nil
}

func parseUserIDs(value string) ([]string, error) {
	seen := make(map[string]struct{})
	result := make([]string, 0)

	for _, item := range strings.Split(value, ",") {
		userID := strings.TrimSpace(item)
		if userID == "" {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}

		seen[userID] = struct{}{}
		result = append(result, userID)
	}

	if len(result) == 0 {
		return nil, status.Error(
			codes.InvalidArgument,
			"at least one user id is required",
		)
	}

	if len(result) > maxBatchUserIDs {
		return nil, status.Error(
			codes.InvalidArgument,
			fmt.Sprintf(
				"at most %d user ids are allowed",
				maxBatchUserIDs,
			),
		)
	}
	return result, nil
}
