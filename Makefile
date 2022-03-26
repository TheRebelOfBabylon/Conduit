PKG := github.com/TheRebelOfBabylon/Conduit

GOBUILD := GO111MODULE=on go build -v
GOINSTALL := GO111MODULE=on go install -v

# ============
# INSTALLATION
# ============

build:
	$(GOBUILD) -tags="${tags}" -o conduit-debug $(PKG)/cmd/conduit
	$(GOBUILD) -tags="${tags}" -o conduitcli-debug $(PKG)/cmd/conduitcli

install:
	$(GOINSTALL) -tags="${tags}" $(PKG)/cmd/conduit
	$(GOINSTALL) -tags="${tags}" $(PKG)/cmd/conduitcli