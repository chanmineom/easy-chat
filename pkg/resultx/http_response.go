package resultx

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func Success(data any) *Response {
	return &Response{
		Code: http.StatusOK,
		Msg:  "",
		Data: data,
	}
}

func Fail(code int, message string) *Response {
	return &Response{
		Code: code,
		Msg:  message,
		Data: nil,
	}
}

func OKHandler(_ context.Context, value any) any {
	return Success(value)
}

func ErrHandler(service string) func(
	context.Context,
	error,
) (int, any) {
	return func(ctx context.Context, err error) (int, any) {
		grpcStatus, ok := status.FromError(err)
		if !ok {
			logx.WithContext(ctx).Errorf(
				"[%s] unexpected error: %v",
				service,
				err,
			)
			return http.StatusInternalServerError,
				Fail(
					http.StatusInternalServerError,
					"internal server error",
				)
		}

		httpCode := grpcCodeToHTTP(grpcStatus.Code())
		message := grpcStatus.Message()

		if httpCode == http.StatusInternalServerError {
			logx.WithContext(ctx).Errorf(
				"[%s] internal rpc error: %v",
				service,
				err,
			)
			message = "internal server error"
		}

		return httpCode, Fail(httpCode, message)
	}
}

func grpcCodeToHTTP(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest

	case codes.Unauthenticated:
		return http.StatusUnauthorized

	case codes.PermissionDenied:
		return http.StatusForbidden

	case codes.NotFound:
		return http.StatusNotFound

	case codes.AlreadyExists:
		return http.StatusConflict

	case codes.FailedPrecondition:
		return http.StatusUnprocessableEntity

	case codes.ResourceExhausted:
		return http.StatusTooManyRequests

	case codes.Unavailable:
		return http.StatusServiceUnavailable

	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout

	case codes.Unimplemented:
		return http.StatusNotImplemented

	default:
		return http.StatusInternalServerError
	}
}
