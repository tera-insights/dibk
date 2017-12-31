# dibk

Disk Image Backup

## Dependencies

We use [`glide`](https://github.com/Masterminds/glide) for dependency management. `sqlite3` must be installed to run the [tests](#Testing).

## Building

`make dibk`

## Usage

Make sure the `DIBK_CONFIG` environment variable is set. It should contain a (possibly relative) path to a config file that has the options shown in `test/dibk_config.json`.

```
./dibk store --name OBJECT_NAME --input INPUT_FILE
./dibk retrieve --name OBJECT_NAME --version 1 --output OUTPUT_FILE
./dibk help
./dibk --version
```

## Testing

`make test`