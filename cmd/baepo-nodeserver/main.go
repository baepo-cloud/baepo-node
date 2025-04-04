package main

import (
	"github.com/baepo-app/baepo-node/pkg/nodeserver"
	"github.com/baepo-app/baepo-node/pkg/nodeservice"
	_ "github.com/joho/godotenv/autoload"
	"log/slog"
	"net"
	"net/http"
	"os"
)

func main() {
	service, err := nodeservice.New(
		"http://localhost:3000",
		"./resources/cloud-hypervisor",
		"./tmp",
		"./resources/vmlinux",
		"./resources/initramfs.cpio.gz",
		"vg_sandbox",
		"code_interpreter",
	)
	if err != nil {
		panic(err)
	}

	server := nodeserver.New(service)
	httpServer := &http.Server{
		Handler: server.Handler(),
		Addr:    os.Getenv("NODE_ADDR"),
	}
	if httpServer.Addr == "" {
		httpServer.Addr = ":3000"
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		panic(err)
	}

	slog.Info("http server starting", slog.String("addr", httpServer.Addr), slog.String("service", "http"))
	go httpServer.Serve(ln)

	//slog.Info("http server shutting down", slog.String("service", "http"))
	//					return httpServer.Shutdown(ctx)
}
