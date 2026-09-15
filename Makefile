.PHONY: build test vet fmt clean

build:
	go build -trimpath -o diagnos ./cmd/investigator

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

clean:
	rm -f diagnos