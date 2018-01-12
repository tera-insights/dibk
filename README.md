# edis

Encrypted Disk Image Storage

## Dependencies

We use [`glide`](https://github.com/Masterminds/glide) for dependency management. `sqlite3` must be installed to run the [tests](#Testing).

## Building

`make edis`

## Usage

```
./edis store --db $DB_PATH --mbperblock $BLOCK_SIZE --storage $STORAGE_LOCATION --name $OBJECT_NAME --input $INPUT_FILE
./edis retrieve --db $DB_PATH --storage $STORAGE_LOCATION --name $OBJECT_NAME --latest --output OUTPUT_FILE
./edis help
./edis --version
```

See `./edis store --help` and `./edis store --retrieve` for descriptions of the flags.

## Testing

`make test`