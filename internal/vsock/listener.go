// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package vsock

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/mdlayher/vsock"
)

type listener struct {
	listener net.Listener
	port     uint32
	ctx      context.Context
	config   config
}

// Listener returns a net.Listener implementation for guest-side Firecracker
// vsock connections.
func Listen(ctx context.Context, port uint32) (net.Listener, error) {
	l, err := vsock.Listen(port, nil)
	if err != nil {
		return nil, err
	}

	return listener{
		listener: l,
		port:     port,
		config:   defaultConfig(),
		ctx:      ctx,
	}, nil
}

func (l listener) Accept() (net.Conn, error) {
	ctx, cancel := context.WithTimeout(l.ctx, l.config.RetryTimeout)
	defer cancel()

	var attemptCount int
	ticker := time.NewTicker(l.config.RetryInterval)
	defer ticker.Stop()
	tickerCh := ticker.C
	for {
		attemptCount++

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-tickerCh:
			conn, err := tryAccept(l.listener, l.port)
			if isTemporaryNetErr(err) {
				err = fmt.Errorf("temporary vsock accept failure: %w", err)
				continue
			} else if err != nil {
				return nil, fmt.Errorf("non-temporary vsock accept failure: %w", err)
			}

			return conn, nil
		}
	}
}

func (l listener) Close() error {
	return l.listener.Close()
}

func (l listener) Addr() net.Addr {
	return l.listener.Addr()
}

// tryAccept attempts to accept a single host-side connection from the provided
// guest-side listener at the provided port.
func tryAccept(listener net.Listener, port uint32) (net.Conn, error) {
	conn, err := listener.Accept()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()

	return conn, nil
}
