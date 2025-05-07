package logservice

import (
	"bytes"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	"io"
	"sync"
	"time"
)

type logWriter struct {
	service       *Service
	containerName string
	fd            uint32
	buf           bytes.Buffer
	mu            sync.Mutex
}

var _ io.Writer = (*logWriter)(nil)

func (s *Service) NewContainerLogWriter(config coretypes.InitContainerConfig) (io.Writer, io.Writer) {
	stdout := &logWriter{
		service:       s,
		containerName: config.Name,
		fd:            1,
	}
	stderr := &logWriter{
		service:       s,
		containerName: config.Name,
		fd:            2,
	}
	return stdout, stderr
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf.Write(p)

	for {
		line, err := w.buf.ReadString('\n')
		if err == io.EOF {
			w.buf.WriteString(line)
			break
		} else if err != nil {
			return 0, err
		}

		entry := &types.LogEntry{
			Timestamp:     time.Now(),
			ContainerName: w.containerName,
			Fd:            w.fd,
			Content:       line,
		}
		if err = w.service.Write(entry); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}
