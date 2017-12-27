package dibk

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // Needed for Gorm
)

const DBType = "sqlite3"
const DBName = "TEST_DB"
const JunkFileSizeInMB = 2
const BlockSizeInKB = 1024

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
		t.Fatal(err)
	}

	err = writeToJunkFile(file)
	if err != nil {
		t.Fatal(err)
	}

	e := Engine{
		db:              testDB,
		blockSizeInKB:   BlockSizeInKB,
		storageLocation: os.TempDir(),
	}
	version := 1
	err = e.saveObject(file, objectName, version)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := e.loadBlockInfos(objectName, version)
	if err != nil {
		t.Fatal(err)
	}

	correctChecksum, err := getChecksumForPath(path, JunkFileSizeInMB*1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	computedChecksum, err := getChecksumForBlocks(loaded)
	if err != nil {
		t.Fatal(err)
	}

	if computedChecksum != correctChecksum {
		t.Fatal("Checksums were not equal")
	}

	os.Remove(path)
}

func createFile() (string, string, *os.File, error) {
	fileName := "dummy_file_" + strconv.Itoa(rand.Int())
	filePath := path.Join(os.TempDir(), fileName)
	for !isFileNew(filePath) {
		fileName = "dummy_file_" + strconv.Itoa(rand.Int())
		filePath = path.Join(os.TempDir(), fileName)
	}
	file, err := os.Create(filePath)
	return fileName, filePath, file, err
}

func writeToJunkFile(file *os.File) error {
	p := make([]byte, JunkFileSizeInMB*1024*1024)
	_, err := rand.Read(p)
	if err != nil {
		return err
	}

	_, err = file.Write(p)
	return err
}

func getChecksumForBlocks(blocks []Block) (string, error) {
	blockSizeInBytes := BlockSizeInKB * 1024
	p := make([]byte, JunkFileSizeInMB*1024*1024)
	for i := 0; i < len(blocks); i++ {
		q, err := read(blocks[i].Location, blockSizeInBytes)
		if err != nil {
			return "", err
		}
		baseIndex := blocks[i].BlockIndex * blockSizeInBytes
		for j := 0; j < blockSizeInBytes; j++ {
			p[baseIndex+j] = q[j]
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256(p)), nil
}

func setup() error {
	rand.Seed(time.Now().UTC().UnixNano())

	db, err := gorm.Open(DBType, DBName)
	if err != nil {
		return err
	}

	testDB = db
	err = testDB.AutoMigrate(&ObjectVersion{}).Error
	if err != nil {
		return err
	}

	err = testDB.AutoMigrate(&Block{}).Error
	return err
}

func teardown() error {
	err := testDB.Close()
	if err != nil {
		return err
	}

	if DBType == "sqlite3" {
		return os.Remove(DBName)
	}
	return nil
}
