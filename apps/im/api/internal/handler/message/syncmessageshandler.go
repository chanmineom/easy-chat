// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package message

import (
	"net/http"

	"github.com/chanmineom/easy-chat/apps/im/api/internal/logic/message"
	"github.com/chanmineom/easy-chat/apps/im/api/internal/svc"
	"github.com/chanmineom/easy-chat/apps/im/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// sync messages after a conversation sequence
func SyncMessagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SyncMessagesReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := message.NewSyncMessagesLogic(r.Context(), svcCtx)
		resp, err := l.SyncMessages(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
