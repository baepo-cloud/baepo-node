package connecthandler

import (
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/initd/v1/initdv1pbconnect"
)

type InitServiceHandler struct {
	init types.InitService
}

var _ initdv1pbconnect.InitDHandler = (*InitServiceHandler)(nil)

func NewInitDServiceServer(init types.InitService) *InitServiceHandler {
	return &InitServiceHandler{init: init}
}
