.PHONY: build release test vet fmt clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o diagnos ./cmd/investigator

# Produces the two files needed by a downloaded release: diagnos and app.yaml.
# Set VERSION=v1.2.3 when cutting a tagged release.
release:
	mkdir -p dist
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/diagnos ./cmd/investigator
	cp app.yaml dist/app.yaml
	chmod 0755 dist/diagnos

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

clean:
	rm -rf dist
	rm -f diagnos
