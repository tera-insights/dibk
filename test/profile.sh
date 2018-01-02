go build $PATH_TO_EXECUTABLE

dd bs=1M count=512 if=/dev/urandom of=large_binary status=none
./dibk --profile /tmp/store_large_binary.prof store --name a --input large_binary
rm large_binary
rm TEST_DB
go tool pprof dibk /tmp/store_large_binary.prof