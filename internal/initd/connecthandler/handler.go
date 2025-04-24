package connecthandler

import (
	"github.com/baepo-cloud/baepo-node/internal/initd/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/initd/v1/initdv1pbconnect"
)

type InitDServiceHandler struct {
	init types.InitD
}

var _ initdv1pbconnect.InitDHandler = (*InitDServiceHandler)(nil)

func NewInitDServiceServer(init types.InitD) *InitDServiceHandler {
	return &InitDServiceHandler{init: init}
}
