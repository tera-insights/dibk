package dibk

// Block is the Gorm model that represents a single block of the file.
type Block struct {
	SHA256Checksum string
	Location       string
	BlockIndex     int // 0-based
	Version        int
	ObjectID       string
}

// ObjectVersion represents a version of a binary object.
type ObjectVersion struct {
	Name           string `gorm:"primary_key"`
	Version        int
	NumberOfBlocks int
}
