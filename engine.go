package dibk

import (
	"fmt"
	"math"
	"os"
	"path"
	"strconv"

	"github.com/ncw/directio"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // for gorm
)

// Configuration defines the paths and variables needed to run dibk.
type Configuration struct {
	DBPath            string
	StorageLocation   string
	IsDirectIOEnabled bool
}

// Engine interacts with the database.
type Engine struct {
	db *gorm.DB
	c  Configuration
}

// MakeEngine onnects to the specified DB and runs the `AutoMigrate` steps.
func MakeEngine(c Configuration) (Engine, error) {
	db, err := gorm.Open("sqlite3", c.DBPath)
	if err != nil {
		return Engine{}, err
	}

	err = db.AutoMigrate(ObjectVersion{}).Error
	if err != nil {
		return Engine{}, err
	}

	err = db.AutoMigrate(Block{}).Error
	return Engine{
		db: db,
		c:  c,
	}, err
}

func (e *Engine) getObjectVersion(name string, version int) (ObjectVersion, error) {
	ov := ObjectVersion{}
	err := e.db.First(&ov, &ObjectVersion{
		Name:    name,
		Version: version,
	}).Error
	if err != nil {
		return ov, err
	}
	return ov, nil
}

func (e *Engine) getAllBlocks(name string) ([]Block, error) {
	var allBlocks []Block
	err := e.db.Find(&allBlocks, &Block{
		ObjectName: name,
	}).Error
	if err != nil {
		return []Block{}, err
	}
	return allBlocks, nil
}

func getLatestBlocks(ov ObjectVersion, all []Block) ([]Block, error) {
	latest := make([]Block, ov.NumberOfBlocks)
	written := make([]bool, ov.NumberOfBlocks)
	for i := 0; i < len(all); i++ {
		current := all[i]
		isRelevant := current.Version <= ov.Version && current.BlockIndex < ov.NumberOfBlocks
		if !isRelevant {
			continue
		}

		old := latest[current.BlockIndex]
		isNewer := old == Block{} || old.Version < current.Version
		if isNewer {
			latest[current.BlockIndex] = current
			written[current.BlockIndex] = true
		}
	}

	for i := 0; i < len(written); i++ {
		if !written[i] {
			return latest, fmt.Errorf("Couldn't find block %d", i)
		}
	}

	return latest, nil
}

func (e *Engine) loadLatestBlock(name string, index int) (Block, error) {
	var found []Block
	err := e.db.Model(&Block{}).Find(&found, &Block{
		BlockIndex: index,
		ObjectName: name,
	}).Error
	if err != nil {
		return Block{}, err
	}

	if len(found) == 0 {
		return Block{}, fmt.Errorf("Did not fetch any blocks for object %s at index index %d", name, index)
	}

	latest := found[0]
	for i := 1; i < len(found); i++ {
		if found[i].Version > latest.Version {
			latest = found[i]
		}
	}
	return latest, nil
}

func (e *Engine) getLatestVersion(name string) (ObjectVersion, error) {
	var found []ObjectVersion
	err := e.db.Model(&ObjectVersion{}).Find(&found, &ObjectVersion{
		Name: name,
	}).Error

	if err != nil {
		return ObjectVersion{}, err
	}

	if len(found) == 0 {
		return ObjectVersion{}, fmt.Errorf("Could not find any objects with name %s", name)
	}

	latest := found[0]
	for i := 1; i < len(found); i++ {
		if found[i].Version > latest.Version {
			latest = found[i]
		}
	}
	return latest, nil
}

func (e *Engine) loadBlockInfos(objectID string, version int) ([]Block, error) {
	ov, err := e.getObjectVersion(objectID, version)
	if err != nil {
		return []Block{}, err
	}

	all, err := e.getAllBlocks(objectID)
	if err != nil {
		return []Block{}, err
	}

	return getLatestBlocks(ov, all)
}

func (e *Engine) getBlockInFile(source *os.File, blockSize, index int) ([]byte, error) {
	offset := int64(blockSize * index)
	info, err := source.Stat()
	if err != nil {
		return []byte{}, err
	}
	remainingBytes := info.Size() - int64(blockSize*index)
	bufferSize := int(math.Min(float64(remainingBytes),
		float64(blockSize)))
	p := make([]byte, bufferSize)
	_, err = source.ReadAt(p, offset)
	return p, err
}

func (e *Engine) getNumBlocksInFile(file *os.File, blockSize int) (int, error) {
	stat, err := file.Stat()
	if err != nil {
		return -1, err
	}
	return int(math.Ceil(float64(stat.Size()) / float64(blockSize))), nil
}

func (e *Engine) isObjectNew(name string) (bool, error) {
	var count int64
	err := e.db.Model(&ObjectVersion{}).Where(&ObjectVersion{
		Name: name,
	}).Count(&count).Error
	return count == 0, err
}

func (e *Engine) shouldWriteBlock(ov ObjectVersion, blockIndex int, newBlockChecksum string) (bool, error) {
	isObjectNew, err := e.isObjectNew(ov.Name)
	if err != nil {
		return false, err
	}

	if isObjectNew {
		return true, nil
	}

	latestVersion, err := e.getLatestVersion(ov.Name)
	if err != nil {
		return false, err
	}

	isBlockNew := blockIndex >= latestVersion.NumberOfBlocks
	if isBlockNew {
		return true, nil
	}

	latestBlock, err := e.loadLatestBlock(ov.Name, blockIndex)
	if err != nil {
		return false, err
	}

	isBlockChanged := latestBlock.SHA1Checksum != newBlockChecksum
	return isBlockChanged, nil
}

func (e *Engine) openFileWithMode(inputPath string, mode int) (*os.File, error) {
	if e.c.IsDirectIOEnabled {
		return directio.OpenFile(inputPath, mode, 0666)
	}
	return os.OpenFile(inputPath, mode, 0666)
}

// OpenFileForReading opens a file for reading, possibly using directIO.
func (e *Engine) OpenFileForReading(inputPath string) (*os.File, error) {
	return e.openFileWithMode(inputPath, os.O_RDONLY)
}

// CreateFileForWriting creates a file for writing, possibly using directIO.
func (e *Engine) CreateFileForWriting(inputPath string) (*os.File, error) {
	return e.openFileWithMode(inputPath, os.O_CREATE|os.O_WRONLY)
}

func (e *Engine) writeBytesAsBlock(ov ObjectVersion, blockNumber int, p []byte) (string, error) {
	blockName := ov.Name + "-" + strconv.Itoa(ov.Version) + "-" + strconv.Itoa(blockNumber) + ".dibk"
	path := path.Join(e.c.StorageLocation, blockName)
	if !isFileNew(path) {
		return path, fmt.Errorf("Block with name %s already exists", path)
	}

	if e.c.IsDirectIOEnabled && len(p)%directio.BlockSize != 0 {
		return "", fmt.Errorf("Passed buffer was not a multilpe of the directio block size\n")
	}

	f, err := e.CreateFileForWriting(path)
	if err != nil {
		return path, err
	}

	_, err = f.Write(p)
	if err != nil {
		return path, err
	}

	err = f.Close()
	return path, err
}

func (e *Engine) getNextVersionNumber(name string) (int, error) {
	var count int
	err := e.db.Model(&ObjectVersion{}).Where(&ObjectVersion{
		Name: name,
	}).Count(&count).Error

	if err != nil {
		return -1, err
	}

	if count == 0 {
		return 1, nil
	}

	ov, err := e.getLatestVersion(name)
	if err != nil {
		return -1, err
	}

	return ov.Version + 1, nil
}

// RetrieveLatestVersionOfObject writes the latest version of the object with the given name to the given file. If the object does not exist or any other errors occur, returns an error.
func (e *Engine) RetrieveLatestVersionOfObject(file *os.File, name string) error {
	ov, err := e.getLatestVersion(name)
	if err != nil {
		return err
	}

	return e.RetrieveObject(file, name, ov.Version)
}

// RetrieveObject retrieves a particular object version.
func (e *Engine) RetrieveObject(file *os.File, name string, version int) error {
	var count int64
	var ov ObjectVersion
	err := e.db.Model(&ObjectVersion{}).Where(&ObjectVersion{
		Name:    name,
		Version: version,
	}).Count(&count).First(&ov).Error

	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("Cannot retrieve object that doesn't exist")
	}

	blocks, err := e.loadBlockInfos(name, version)
	if err != nil {
		return err
	}

	for i := 0; i < len(blocks); i++ {
		info, err := os.Stat(blocks[i].Location)
		if err != nil {
			return err
		}

		p, err := read(blocks[i].Location, int(info.Size()))
		if err != nil {
			return err
		}

		offset := int64(ov.BlockSize * blocks[i].BlockIndex)
		n, err := file.WriteAt(p, offset)
		if n != len(p) {
			return fmt.Errorf("Did not write all bytes from block")
		}
	}

	return nil
}

func (e *Engine) makeNewerObjectVersion(file *os.File, name string, blockSize int) (ObjectVersion, error) {
	nextVersion, err := e.getNextVersionNumber(name)
	if err != nil {
		return ObjectVersion{}, err
	}

	nBlocks, err := e.getNumBlocksInFile(file, blockSize)
	if err != nil {
		return ObjectVersion{}, err
	}

	return ObjectVersion{
		Name:           name,
		Version:        nextVersion,
		NumberOfBlocks: nBlocks,
		BlockSize:      blockSize,
	}, nil
}

func (e *Engine) saveObjectAndBlocksInDatabase(ov ObjectVersion, results []blockWriteResult) error {
	err := e.db.Create(&ov).Error
	if err != nil {
		return err
	}

	tx := e.db.Begin()
	for i := 0; i < len(results); i++ {
		if results[i].isNew {
			b := Block{
				SHA1Checksum: results[i].checksum,
				Location:     results[i].path,
				BlockIndex:   results[i].blockNumber,
				ObjectName:   ov.Name,
				Version:      ov.Version,
			}
			err = tx.Create(&b).Error
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

// SaveObject saves a binary object.
func (e *Engine) SaveObject(file *os.File, name string, blockSize int) error {
	ov, err := e.makeNewerObjectVersion(file, name, blockSize)
	if err != nil {
		return err
	}

	results, err := makeFileWriterWorkerPool(e, ov, file, e.c.IsDirectIOEnabled).
		write()
	if err != nil {
		return err
	}

	return e.saveObjectAndBlocksInDatabase(ov, results)
}

func read(path string, sizeInBytes int) ([]byte, error) {
	p := make([]byte, sizeInBytes)
	file, err := os.Open(path)
	if err != nil {
		return p, err
	}

	n, err := file.Read(p)
	if n != sizeInBytes {
		return p, fmt.Errorf("Did not read enough data from file")
	} else if err != nil {
		return p, err
	}
	return p, nil
}

func isFileNew(path string) bool {
	_, err := os.Stat(path)
	return !os.IsExist(err)
}
