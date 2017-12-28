# dibk

Disk Image Backup

## Building

`go build cmd/dibk.go`

## Usage

Make sure the `DIBK_CONFIG` environment variable is set. It should contain a (possibly relative) path to a config file that has the options shown in `test/dibk_config.json`.

```
./dibk store --name OBJECT_NAME --input INPUT_FILE
./dibk retrieve --name OBJECT_NAME --version 1 --output OUTPUT_FILE
./dibk help
./dibk --version
```

## Testing

Must have `sqlite3` installed and available as an executable under that name.

To run the Go tests, use ` go test `.

Alternatively, to run the CLI tests, use `PATH_TO_EXECUTABLE=cmd/dibk.go DIBK_CONFIG=test/dibk_config.json ./test/cli_test.sh`