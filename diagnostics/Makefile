# Old-skool build tools.
#
# Targets (see each target for more information):
#   build: Build code.

GODEP_DIR = Godeps/_workspace

# Build code.
#
# Example:
#   make
#   make build
all build:
	GOPATH=`pwd`/${GODEP_DIR}:${GOPATH} go build -o diagnostics main.go
