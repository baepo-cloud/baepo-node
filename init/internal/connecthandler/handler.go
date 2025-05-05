package connecthandler

import (
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
)

type InitServiceHandler struct {
	init types.InitService
}

var _ nodev1pbconnect.InitHandler = (*InitServiceHandler)(nil)

func NewInitServiceServer(init types.InitService) *InitServiceHandler {
	return &InitServiceHandler{init: init}
}
