package connecthandler

import (
	"github.com/baepo-app/baepo-node/pkg/initd/types"
	"github.com/baepo-app/baepo-node/pkg/proto/v1/v1connect"
)

type InitDServiceHandler struct {
	init types.InitD
}

var _ v1connect.InitDHandler = (*InitDServiceHandler)(nil)

func NewInitDServiceServer(init types.InitD) *InitDServiceHandler {
	return &InitDServiceHandler{init: init}
}
