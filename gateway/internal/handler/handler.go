package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/status"
)

func errResp(c *gin.Context, err error) {
	if st, ok := status.FromError(err); ok {
		code := grpcToHTTP(st.Code())
		c.JSON(code, gin.H{"error": st.Message()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func grpcToHTTP(code interface{ String() string }) int {
	switch code.String() {
	case "NotFound":
		return http.StatusNotFound
	case "AlreadyExists":
		return http.StatusConflict
	case "InvalidArgument":
		return http.StatusBadRequest
	case "PermissionDenied":
		return http.StatusForbidden
	case "Unauthenticated":
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func intQuery(c *gin.Context, key string, def int) int32 {
	v, err := strconv.Atoi(c.DefaultQuery(key, strconv.Itoa(def)))
	if err != nil {
		return int32(def)
	}
	return int32(v)
}
