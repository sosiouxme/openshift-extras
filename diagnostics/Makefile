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
	GOPATH=${GOPATH}:`pwd`/${GODEP_DIR} go build
