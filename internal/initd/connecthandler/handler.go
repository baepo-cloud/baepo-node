package connecthandler

import (
	"github.com/baepo-cloud/baepo-node/internal/initd/types"
	"github.com/baepo-cloud/baepo-node/pkg/proto/baepo/node/v1/v1connect"
)

type InitDServiceHandler struct {
	init types.InitD
}

var _ v1connect.InitDHandler = (*InitDServiceHandler)(nil)

func NewInitDServiceServer(init types.InitD) *InitDServiceHandler {
	return &InitDServiceHandler{init: init}
}
