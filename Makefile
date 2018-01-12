.PHONY: test clean

edis: vendor/ *.go cmd/*.go
	go build cmd/edis.go

test: vendor/
	go test
	PATH_TO_EXECUTABLE=cmd/edis.go ./test/cli_test.sh

vendor/: glide.lock glide.yaml
	glide install
	go install edis/vendor/github.com/mattn/go-sqlite3
	go install edis/vendor/github.com/spacemonkeygo/openssl

clean:
	rm -r vendor
	rm edis