package initserver

import (
	"connectrpc.com/connect"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s InitServiceServer) Healthcheck(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
	//if h.init.MainCmd() == nil || h.init.MainCmd().Process == nil {
	//	return nil, fmt.Errorf("process not found")
	//}
	//
	//process := h.init.MainCmd().Process
	//p, err := os.FindProcess(process.Pid)
	//if err != nil {
	//	return nil, err
	//}

	//return connect.NewResponse(&emptypb.Empty{}), p.Signal(syscall.Signal(0))
	return connect.NewResponse(&emptypb.Empty{}), nil
}
