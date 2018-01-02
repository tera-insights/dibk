dibk: vendor/ *.go cmd/*.go
	go build cmd/dibk.go

test: vendor/
	go test
	PATH_TO_EXECUTABLE=cmd/dibk.go DIBK_CONFIG=test/dibk_config.json ./test/cli_test.sh

vendor/: glide.lock glide.yaml
	glide install
	go install dibk/vendor/github.com/mattn/go-sqlite3
	go install dibk/vendor/github.com/spacemonkeygo/openssl

clean:
	rm -r vendor
	rm dibk