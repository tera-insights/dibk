package dibk

import (
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // for gorm
	"github.com/spacemonkeygo/openssl"
)

// Configuration defines the paths and variables needed to run dibk.
type Configuration struct {
	DBPath          string `json:"db_path"`
	BlockSizeInKB   int    `json:"block_size_in_kb"`
	StorageLocation string `json:"storage_location"`
}

// Engine interacts with the database.
type Engine struct {
	db              *gorm.DB
	blockSizeInKB   int
	storageLocation string
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
		db:              db,
		blockSizeInKB:   c.BlockSizeInKB,
		storageLocation: c.StorageLocation,
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

func (e *Engine) getBlockInFile(source *os.File, index int) ([]byte, error) {
	offset := int64(e.blockSizeInKB * 1024 * index)
	info, err := source.Stat()
	if err != nil {
		return []byte{}, err
	}
	remainingBytes := info.Size() - int64(e.blockSizeInKB*1024*index)
	bufferSize := int(math.Min(float64(remainingBytes),
		float64(e.blockSizeInKB*1024)))
	p := make([]byte, bufferSize)
	_, err = source.ReadAt(p, offset)
	return p, err
}

func (e *Engine) writeBlock(source *os.File, id string, version int, index int) (string, error) {
	blockName := id + "-" + strconv.Itoa(version) + "-" + strconv.Itoa(index) + ".dibk"
	path := path.Join(e.storageLocation, blockName)
	if !isFileNew(path) {
		return path, fmt.Errorf("Block with name %s already exists", path)
	}

	p, err := e.getBlockInFile(source, index)
	if err != nil {
		return path, err
	}

	f, err := os.Create(path)
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

func (e *Engine) getNumBlocksInFile(file *os.File) (int, error) {
	stat, err := file.Stat()
	if err != nil {
		return -1, err
	}
	return int(math.Ceil(float64(stat.Size()) / float64(e.blockSizeInKB*1024))), nil
}

func (e *Engine) isObjectNew(name string) (bool, error) {
	var count int64
	err := e.db.Model(&ObjectVersion{}).Where(&ObjectVersion{
		Name: name,
	}).Count(&count).Error
	return count == 0, err
}

func (e *Engine) shouldWriteBlock(name string, version, blockIndex int, p []byte) (bool, error) {
	isObjectNew, err := e.isObjectNew(name)
	if err != nil {
		return false, err
	}

	if isObjectNew {
		return true, nil
	}

	latestVersion, err := e.getLatestVersion(name)
	if err != nil {
		return false, err
	}

	isBlockNew := blockIndex >= latestVersion.NumberOfBlocks
	if isBlockNew {
		return true, nil
	}

	latestBlock, err := e.loadLatestBlock(name, blockIndex)
	if err != nil {
		return false, err
	}

	hash, err := openssl.SHA1(p)
	if err != nil {
		return false, err
	}

	isBlockChanged := latestBlock.SHA1Checksum != fmt.Sprintf("%x", hash)
	return isBlockChanged, nil
}

func (e *Engine) writeBytesAsBlock(id string, version int, blockNumber int, p []byte) (string, error) {
	blockName := id + "-" + strconv.Itoa(version) + "-" + strconv.Itoa(blockNumber) + ".dibk"
	path := path.Join(e.storageLocation, blockName)
	if !isFileNew(path) {
		return path, fmt.Errorf("Block with name %s already exists", path)
	}

	f, err := os.Create(path)
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

type writeTask struct {
	blockNumber int
	buffer      []byte
}

func (e *Engine) writeFileInBlocks(file *os.File, id string, version int) ([]writeResult, error) {
	agenda := make(chan writeTask)
	finished := make(chan writeResult)
	nBlocks, err := e.getNumBlocksInFile(file)
	if err != nil {
		return []writeResult{}, err
	}
	go func() {
		for true {
			task := <-agenda
			localBuffer := make([]byte, len(task.buffer))
			copy(localBuffer, task.buffer)
			go func() {
				agenda <- task
			}()
			shouldWrite, err := e.shouldWriteBlock(id, version, task.blockNumber, localBuffer)
			if err != nil {
				panic(err)
			}

			if shouldWrite {
				path, err := e.writeBytesAsBlock(id, version, task.blockNumber, localBuffer)
				if err != nil {
					panic(err)
				}
				go func() {
					finished <- writeResult{path, true, task.blockNumber}
				}()
			} else {
				go func() {
					finished <- writeResult{"", false, task.blockNumber}
				}()
			}

			if task.blockNumber == nBlocks-1 {
				break
			}
		}
	}()

	bufferSize := e.blockSizeInKB * 1024
	buffer := make([]byte, bufferSize)
	blockNumber := 0

	_, err = file.Seek(0, 0)
	if err != nil {
		return []writeResult{}, err
	}

	for blockNumber < nBlocks {
		if blockNumber > 0 {
			<-agenda // wait until previous task is complete
		}

		if blockNumber < nBlocks-1 {
			_, err = file.Read(buffer)
		} else {
			buffer, err = e.getBlockInFile(file, blockNumber) // last block may not be full
		}

		if err != nil {
			return []writeResult{}, err
		}
		agenda <- writeTask{blockNumber, buffer}
		blockNumber++
	}

	wg := sync.WaitGroup{}
	wg.Add(nBlocks)

	var wr []writeResult
	go func() {
		for len(wr) < nBlocks {
			x := <-finished
			wr = append(wr, x)
			wg.Done()
		}
	}()

	wg.Wait()
	return wr, nil
}

type writeResult struct {
	path        string
	isNew       bool
	blockNumber int
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

// RetrieveObject retrieves a particular object version.
func (e *Engine) RetrieveObject(file *os.File, name string, version int) error {
	var count int64
	err := e.db.Model(&ObjectVersion{}).Where(&ObjectVersion{
		Name:    name,
		Version: version,
	}).Count(&count).Error

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

		offset := int64(e.blockSizeInKB * 1024 * blocks[i].BlockIndex)
		n, err := file.WriteAt(p, offset)
		if n != len(p) {
			return fmt.Errorf("Did not write all bytes from block")
		}
	}

	return nil
}

// SaveObject saves a binary object.
func (e *Engine) SaveObject(file *os.File, name string) error {
	nextVersion, err := e.getNextVersionNumber(name)
	if err != nil {
		return err
	}

	results, err := e.writeFileInBlocks(file, name, nextVersion)
	if err != nil {
		return err
	}

	err = e.db.Create(&ObjectVersion{
		Name:           name,
		Version:        nextVersion,
		NumberOfBlocks: len(results),
	}).Error
	if err != nil {
		return err
	}

	for i := 0; i < len(results); i++ {
		if !results[i].isNew {
			continue
		}

		info, err := os.Stat(results[i].path)
		if err != nil {
			return err
		}

		checksum, err := getChecksumForPath(results[i].path, int(info.Size()))
		if err != nil {
			return err
		}

		b := Block{
			SHA1Checksum: checksum,
			Location:     results[i].path,
			BlockIndex:   results[i].blockNumber,
			ObjectName:   name,
			Version:      nextVersion,
		}
		err = e.db.Create(&b).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func getChecksumForPath(path string, fileSizeInBytes int) (string, error) {
	p, err := read(path, fileSizeInBytes)
	hash, err := openssl.SHA1(p)
	return fmt.Sprintf("%x", hash), err
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
