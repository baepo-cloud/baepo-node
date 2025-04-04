package nodeserver

import (
	"github.com/baepo-app/baepo-node/pkg/nodeservice"
	"github.com/baepo-app/baepo-node/pkg/proto/v1/v1connect"
	"net/http"
)

type Server struct {
	service *nodeservice.Service
}

var _ v1connect.NodeServiceHandler = (*Server)(nil)

func New(service *nodeservice.Service) *Server {
	return &Server{service: service}
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle(v1connect.NewNodeServiceHandler(s))
	return mux
}
