package dibk

import (
	"crypto/sha256"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"

	"github.com/jinzhu/gorm"
)

// Engine interacts with the database.
type Engine struct {
	db              *gorm.DB
	blockSizeInKB   int
	storageLocation string
}

func (e *Engine) getObjectVersion(objectID string, version int) (ObjectVersion, error) {
	ov := ObjectVersion{}
	err := e.db.First(&ov, &ObjectVersion{
		ID:      objectID,
		Version: version,
	}).Error
	if err != nil {
		return ov, err
	}
	return ov, nil
}

func (e *Engine) getAllBlocks(objectID string) ([]Block, error) {
	allBlocks := make([]Block, 0)
	err := e.db.Find(&allBlocks, &Block{
		ObjectID: objectID,
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
		isRelevant := current.Version <= ov.Version
		if !isRelevant {
			continue
		}

		old := latest[current.BlockIndex]
		isNewer := old == Block{} || old.Version < current.Version
		if isNewer {
			latest[current.BlockIndex] = current
			written[i] = true
		}
	}

	for i := 0; i < len(written); i++ {
		if !written[i] {
			return latest, fmt.Errorf("Couldn't find block %d", i)
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

func (e *Engine) writeBlock(source *os.File, id string, version int, index int) (string, error) {
	blockName := id + "-" + strconv.Itoa(version) + "-" + strconv.Itoa(index) + ".dibk"
	path := path.Join(e.storageLocation, blockName)
	if !isFileNew(path) {
		return path, fmt.Errorf("Block with name %s already exists", path)
	}

	offset := int64(e.blockSizeInKB * 1024 * index)
	p := make([]byte, e.blockSizeInKB*1024)
	_, err := source.ReadAt(p, offset)
	if err != nil {
		return path, err
	}

	f, err := os.Create(path)
	if err != nil {
		return path, err
	}

	n, err := f.Write(p)
	if n != e.blockSizeInKB*1024 {
		return path, fmt.Errorf("Did not write enough bytes")
	} else if err != nil {
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

func (e *Engine) writeFileInBlocks(file *os.File, id string, version int) ([]string, error) {
	nBlocks, err := e.getNumBlocksInFile(file)
	if err != nil {
		return []string{}, err
	}

	paths := make([]string, nBlocks)
	for i := 0; i < nBlocks; i++ {
		path, err := e.writeBlock(file, id, version, i)
		paths[i] = path
		if err != nil {
			return paths, err
		}
	}
	return paths, nil
}

func (e *Engine) saveObject(file *os.File, id string, version int) error {
	var count int
	err := e.db.Model(&ObjectVersion{}).Where(&ObjectVersion{
		ID:      id,
		Version: version,
	}).Count(&count).Error
	if err != nil {
		return err
	}

	isNew := count == 0
	if !isNew {
		return fmt.Errorf("Not a new combination of version and object ID")
	}

	blockPaths, err := e.writeFileInBlocks(file, id, version)
	if err != nil {
		return err
	}

	err = e.db.Create(&ObjectVersion{
		ID:             id,
		Version:        version,
		NumberOfBlocks: len(blockPaths),
	}).Error
	if err != nil {
		return err
	}

	for i := 0; i < len(blockPaths); i++ {
		checksum, err := getChecksumForPath(blockPaths[i], e.blockSizeInKB*1024)
		if err != nil {
			return err
		}

		b := Block{
			SHA256Checksum: checksum,
			Location:       blockPaths[i],
			BlockIndex:     i,
			ObjectID:       id,
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
	return fmt.Sprintf("%x", sha256.Sum256(p)), err
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
