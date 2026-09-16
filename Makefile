.PHONY: build release test vet fmt clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o diagnos ./cmd/investigator

# Produces the two files needed by a downloaded release: diagnos and app.yaml.
# Set VERSION=v1.2.3 when cutting a tagged release.
release:
	rm -rf dist
	mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/diagnos_$(VERSION)_darwin_arm64 ./cmd/investigator
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/diagnos_$(VERSION)_linux_amd64 ./cmd/investigator
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/diagnos_$(VERSION)_linux_arm64 ./cmd/investigator
	cp app.yaml dist/app.yaml
	chmod 0755 dist/diagnos_*

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

clean:
	rm -rf dist
	rm -f diagnos
