# dibk

Disk Image Backup

## Testing

Must have `sqlite3` installed and available as an executable under that name.

To run the Go tests, use ` go test `.

Alternatively, to run the CLI tests, use `PATH_TO_EXECUTABLE=cmd/dibk.go DIBK_CONFIG=test/dibk_config.json ./test/cli_test.sh`