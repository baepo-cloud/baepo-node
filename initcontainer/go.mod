module github.com/baepo-cloud/baepo-node/initcontainer

go 1.24.2

replace github.com/baepo-cloud/baepo-node/core => ../core

require (
	github.com/baepo-cloud/baepo-node/core v0.0.0-00010101000000-000000000000
	golang.org/x/sys v0.32.0
)
