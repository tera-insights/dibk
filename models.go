package edis

// Block is the Gorm model that represents a single block of the file.
type Block struct {
	SHA1Checksum string
	Location     string
	BlockIndex   int    `gorm:"unique_index:block_index_version_object_name"` // 0-based
	Version      int    `gorm:"unique_index:block_index_version_object_name"`
	ObjectName   string `gorm:"unique_index:block_index_version_object_name"`
}

// ObjectVersion represents a version of a binary object.
type ObjectVersion struct {
	Name           string `gorm:"unique_index:id_version"`
	Version        int    `gorm:"unique_index:id_version"`
	BlockSize      int
	NumberOfBlocks int
}
