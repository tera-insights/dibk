package dibk

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // Needed for Gorm
)

const DBType = "sqlite3"
const DBName = "TEST_DB"
const JunkFileSizeInMB = 2
const BlockSizeInKB = 4

var testDB *gorm.DB

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		panic(err)
	}

	result := m.Run()
	teardown()
	os.Exit(result)
}

func TestUploadingAndRetrievingSameFile(t *testing.T) {
	objectName, path, file, err := createFile()
	if err != nil {
		t.Error(err)
	}

	err = writeToJunkFile(file)
	if err != nil {
		t.Error(err)
	}

	e := Engine{testDB}
	version := 1
	err = e.saveObject(objectName, path, version)
	if err != nil {
		t.Error(err)
	}

	loaded, err := e.loadBlockInfos(objectName, version)
	if err != nil {
		t.Error(err)
	}

	correctChecksum, err := getChecksumForPath(path)
	if err != nil {
		t.Error(err)
	}

	computedChecksum, err := getChecksumForBlocks(loaded)
	if err != nil {
		t.Error(err)
	}

	if computedChecksum != correctChecksum {
		t.Error("Checksums were not equal")
	}

	os.Remove(path)
}

func createFile() (string, string, *os.File, error) {
	fileName := "dummy_file_" + string(rand.Int())
	filePath := path.Join(os.TempDir(), fileName)
	for _, err := os.Stat(filePath); !os.IsExist(err); {
		fileName = "dummy_file_" + string(rand.Int())
		filePath = path.Join(os.TempDir(), fileName)
	}
	file, err := os.Create(filePath)
	return fileName, filePath, file, err
}

func writeToJunkFile(file *os.File) error {
	p := make([]byte, JunkFileSizeInMB*1024*1024)
	n, err := rand.Read(p)
	if err != nil {
		return err
	}

	_, err = file.Write(p)
	return err
}

func getChecksumForPath(path string) (string, error) {
	p, err := read(path, JunkFileSizeInMB*1024*1024)
	return fmt.Sprintf("%x", sha256.Sum256(p)), nil
}

func getChecksumForBlocks(blocks []Block) (string, error) {
	blockSizeInBytes := BlockSizeInKB * 1024
	p := make([]byte, JunkFileSizeInMB*1024*1024)
	for i := 0; i < len(blocks); i++ {
		q, err := read(blocks[i].Location, blockSizeInBytes)
		baseIndex := blocks[i].BlockIndex * blockSizeInBytes
		for j := 0; j < blockSizeInBytes; j++ {
			p[baseIndex+j] = q[j]
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256(p)), nil
}

func read(path string, size int) ([]byte, error) {
	p := make([]byte, size)
	file, err := os.Open(path)
	if err != nil {
		return p, err
	}

	n, err := file.Read(p)
	if n != size {
		return p, fmt.Errorf("Did not read enough data from file")
	} else if err != nil {
		return p, err
	}
	return p, nil
}

func setup() error {
	db, err := gorm.Open(DBType, DBName)
	testDB = db
	return err
}

func teardown() error {
	err := testDB.Close()
	if err != nil {
		return err
	}

	if DBType == "sqlite3" {
		return os.Remove(DBName + DBType)
	}
	return nil
}
