package gatewayserver

import (
	"errors"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"net"
	"net/http"
	"net/http/httputil"
)

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		machineID, port, err := net.SplitHostPort(r.Host)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		machine, err := s.service.FindMachine(r.Context(), machineID)
		if errors.Is(err, types.ErrMachineNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if machine.NetworkInterface == nil || machine.State != types.MachineStateRunning {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		targetURL := *r.URL
		targetURL.Scheme = "http"
		targetURL.Host = net.JoinHostPort(machine.NetworkInterface.IPAddress.String(), port)
		proxy := httputil.NewSingleHostReverseProxy(&targetURL)
		proxy.ServeHTTP(w, r)
	})
}
