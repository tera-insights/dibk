.PHONY: test clean

edis: vendor/ *.go cmd/*.go
	go build cmd/edis.go

test: vendor/
	go test
	PATH_TO_EXECUTABLE=cmd/edis.go ./test/cli_test.sh

vendor/: glide.lock glide.yaml
	glide install

clean:
	rm -r vendor
	rm edis
