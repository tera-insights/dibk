# dibk

Disk Image Backup

## Dependencies

We use [`glide`](https://github.com/Masterminds/glide) for dependency management. `sqlite3` must be installed to run the [tests](#Testing).

## Building

`make dibk`

## Usage

```
./dibk store --db $DB_PATH --mbperblock $BLOCK_SIZE --storage $STORAGE_LOCATION --name OBJECT_NAME --input INPUT_FILE
./dibk retrieve --db $DB_PATH --blocksize $BLOCK_SIZE --storage $STORAGE_LOCATION --name OBJECT_NAME --version 1 --output OUTPUT_FILE
./dibk help
./dibk --version
```

## Testing

`make test`