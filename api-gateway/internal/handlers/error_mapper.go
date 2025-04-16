package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapGrpcToHttpError(c *gin.Context, err error, requestInfo string) { // Добавлен requestInfo для логирования
	st, ok := status.FromError(err)
	if !ok {
		log.Printf("API Gateway: Non-gRPC error processing request '%s': %v", requestInfo, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to communicate with downstream service"}) // 502 Bad Gateway
		return
	}

	httpStatus := http.StatusInternalServerError
	errMsg := st.Message()

	log.Printf("API Gateway: gRPC error processing request '%s': Code=%s, Msg=%s", requestInfo, st.Code(), errMsg)

	switch st.Code() {
	case codes.NotFound:
		httpStatus = http.StatusNotFound // 404
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest // 400
	case codes.AlreadyExists:
		httpStatus = http.StatusConflict // 409
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden // 403
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized // 401
	case codes.FailedPrecondition:
		httpStatus = http.StatusConflict // 409
	case codes.Unimplemented:
		httpStatus = http.StatusNotImplemented // 501
	case codes.Unavailable:
		httpStatus = http.StatusBadGateway // 502
	case codes.DeadlineExceeded:
		httpStatus = http.StatusGatewayTimeout // 504
	case codes.Internal:
		httpStatus = http.StatusInternalServerError // 500
		errMsg = "Internal server error in downstream service"
	default:
		httpStatus = http.StatusInternalServerError
		errMsg = "An unexpected error occurred"
	}

	c.JSON(httpStatus, gin.H{"error": errMsg})
}
