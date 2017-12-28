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
const DefaultJunkFileSizeInMB = 2
const BlockSizeInKB = 1024

var testDB *gorm.DB
var e Engine

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
	objectName, path, _, err := createAndSaveJunkFile(1)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := e.loadBlockInfos(objectName, 1)
	if err != nil {
		t.Fatal(err)
	}

	correctChecksum, err := getChecksumForPath(path, DefaultJunkFileSizeInMB*1024*1024)
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

func createAndSaveJunkFile(version int) (objectName string, path string, file *os.File, err error) {
	objectName, path, file, err = createTemporaryFile()
	if err != nil {
		return
	}

	err = writeToJunkFile(file)
	if err != nil {
		return
	}

	err = e.saveObject(file, objectName, 1)
	return
}

func TestChangingBlocksWithSameSizeFile(t *testing.T) {
	objectName, path, file, err := createAndSaveJunkFile(1)
	if err != nil {
		t.Fatal(err)
	}

	correctVersionOneBlocks, err := e.loadBlockInfos(objectName, 1)
	if err != nil {
		t.Fatal(err)
	}

	nBlocks, err := e.getNumBlocksInFile(file)
	if err != nil {
		t.Fatal(err)
	}

	newBytes := make([]byte, nBlocks*BlockSizeInKB*1024)
	oldBytes, err := read(path, nBlocks*BlockSizeInKB*1024)
	if err != nil {
		t.Fatal(err)
	}

	copy(newBytes, oldBytes)

	nToChange := 1
	changedIndices := make([]int, nToChange)
	for i := 0; i < len(changedIndices); i++ {
		changedIndices[i] = -1
		index := rand.Int() % nBlocks
		for !isNew(changedIndices, index) {
			index = rand.Int() % nBlocks
		}
		changedIndices[i] = index
		p := make([]byte, BlockSizeInKB*1024)
		_, err = rand.Read(p)
		for j := 0; j < BlockSizeInKB*1024; j++ {
			offset := index*BlockSizeInKB*1024 + j
			newBytes[offset] = p[j]
		}
	}

	newPath, err := createAndSaveFile(objectName, newBytes, 2)
	if err != nil {
		t.Fatal(err)
	}

	isEqual, err := fetchAndCheck(objectName, 1, correctVersionOneBlocks)
	if err != nil {
		t.Fatal(err)
	}

	if !isEqual {
		t.Fatalf("Original version one blocks did not equal those we just fetched")
	}

	fetchedVersionTwoBlocks, err := e.loadBlockInfos(objectName, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(fetchedVersionTwoBlocks) != nBlocks {
		t.Fatalf("Did not load proper number of blocks for version two")
	}

	for i := 0; i < len(fetchedVersionTwoBlocks); i++ {
		block := fetchedVersionTwoBlocks[i]
		isChangedBlock := false
		for j := 0; j < len(changedIndices); j++ {
			if changedIndices[j] == i {
				isChangedBlock = true
			}
		}
		isCorrectVersion := (isChangedBlock && block.Version == 2) ||
			(!isChangedBlock && block.Version == 1)
		if !isCorrectVersion {
			t.Fatalf("Block versions did not match what was changed")
		}
	}

	fileChecksum, err := getChecksumForPath(newPath, DefaultJunkFileSizeInMB*1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	blocksChecksum, err := getChecksumForBlocks(fetchedVersionTwoBlocks)
	if err != nil {
		t.Fatal(err)
	}

	if fileChecksum != blocksChecksum {
		t.Fatalf("File and block checksums were not equal")
	}
}

// Version 2 has one more block than version one. The new block is appended
// to the end of the file.
func TestNewVersionWithLargerSize(t *testing.T) {
	objectName, path, file, err := createAndSaveJunkFile(1)
	if err != nil {
		t.Fatal(err)
	}

	correctVersionOneBlocks, err := e.loadBlockInfos(objectName, 1)
	if err != nil {
		t.Fatal(err)
	}

	nBlocks, err := e.getNumBlocksInFile(file)
	if err != nil {
		t.Fatal(err)
	}

	newBytes := make([]byte, BlockSizeInKB*1024*(nBlocks+1))
	oldBytes, err := read(path, nBlocks*BlockSizeInKB*1024)
	if err != nil {
		t.Fatal(err)
	}

	copy(newBytes, oldBytes)
	p := make([]byte, BlockSizeInKB*1024)
	_, err = rand.Read(p)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(p); i++ {
		offset := nBlocks * BlockSizeInKB * 1024
		newBytes[offset+i] = p[i]
	}

	newPath, err := createAndSaveFile(objectName, newBytes, 2)
	if err != nil {
		t.Fatal(err)
	}

	isEqual, err := fetchAndCheck(objectName, 1, correctVersionOneBlocks)
	if err != nil {
		t.Fatal(err)
	}

	if !isEqual {
		t.Fatalf("Original version one blocks did not equal those we just fetched")
	}

	fetchedVersionTwoBlocks, err := e.loadBlockInfos(objectName, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(fetchedVersionTwoBlocks) != nBlocks+1 {
		t.Fatalf("Did not load proper number of blocks for version two")
	}

	for i := 0; i < len(fetchedVersionTwoBlocks); i++ {
		block := fetchedVersionTwoBlocks[i]
		isChangedBlock := i == nBlocks
		isCorrectVersion := (isChangedBlock && block.Version == 2) ||
			(!isChangedBlock && block.Version == 1)
		if !isCorrectVersion {
			t.Fatalf("Block versions did not match what was changed")
		}
	}

	fileChecksum, err := getChecksumForPath(newPath, DefaultJunkFileSizeInMB*1024*1024+BlockSizeInKB*1024)
	if err != nil {
		t.Fatal(err)
	}

	blocksChecksum, err := getChecksumForBlocks(fetchedVersionTwoBlocks)
	if err != nil {
		t.Fatal(err)
	}

	if fileChecksum != blocksChecksum {
		t.Fatalf("File and block checksums were not equal")
	}
}

func TestNewVersionWithSmallerSize(t *testing.T) {
	objectName, path, file, err := createAndSaveJunkFile(1)
	if err != nil {
		t.Fatal(err)
	}

	correctVersionOneBlocks, err := e.loadBlockInfos(objectName, 1)
	if err != nil {
		t.Fatal(err)
	}

	nBlocks, err := e.getNumBlocksInFile(file)
	if err != nil {
		t.Fatal(err)
	}

	if nBlocks == 1 {
		t.Fatalf("Cannot create a smaller file that only has one block")
	}

	newBytes := make([]byte, BlockSizeInKB*1024*(nBlocks-1))
	oldBytes, err := read(path, nBlocks*BlockSizeInKB*1024)
	if err != nil {
		t.Fatal(err)
	}

	copy(newBytes, oldBytes)

	newPath, err := createAndSaveFile(objectName, newBytes, 2)
	if err != nil {
		t.Fatal(err)
	}

	isEqual, err := fetchAndCheck(objectName, 1, correctVersionOneBlocks)
	if err != nil {
		t.Fatal(err)
	}

	if !isEqual {
		t.Fatalf("Original version one blocks did not equal those we just fetched")
	}

	fetchedVersionTwoBlocks, err := e.loadBlockInfos(objectName, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(fetchedVersionTwoBlocks) != nBlocks-1 {
		t.Fatalf("Did not load proper number of blocks for version two")
	}

	for i := 0; i < len(fetchedVersionTwoBlocks); i++ {
		block := fetchedVersionTwoBlocks[i]
		isChangedBlock := false
		isCorrectVersion := (isChangedBlock && block.Version == 2) ||
			(!isChangedBlock && block.Version == 1)
		if !isCorrectVersion {
			t.Fatalf("Block versions did not match what was changed")
		}
	}

	fileChecksum, err := getChecksumForPath(newPath, DefaultJunkFileSizeInMB*1024*1024-BlockSizeInKB*1024)
	if err != nil {
		t.Fatal(err)
	}

	blocksChecksum, err := getChecksumForBlocks(fetchedVersionTwoBlocks)
	if err != nil {
		t.Fatal(err)
	}

	if fileChecksum != blocksChecksum {
		t.Fatalf("File and block checksums were not equal")
	}
}

func fetchAndCheck(objectName string, version int, b []Block) (bool, error) {
	fetchedVersionOneBlocks, err := e.loadBlockInfos(objectName, 1)
	if err != nil {
		return false, err
	}

	return isEqual(fetchedVersionOneBlocks, b), nil
}

func createAndSaveFile(objectName string, content []byte, version int) (string, error) {
	_, newPath, newFile, err := createTemporaryFile()
	if err != nil {
		return newPath, err
	}

	_, err = newFile.Write(content)
	if err != nil {
		return newPath, err
	}

	return newPath, e.saveObject(newFile, objectName, 2)
}

func isEqual(a, b []Block) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		isCorrect := (a[i].BlockIndex == b[i].BlockIndex &&
			a[i].Location == b[i].Location &&
			a[i].ObjectName == b[i].ObjectName &&
			a[i].SHA256Checksum == b[i].SHA256Checksum &&
			a[i].Version == b[i].Version)
		if !isCorrect {
			return false
		}
	}

	return true
}

func isNew(a []int, b int) bool {
	for i := 0; i < len(a); i++ {
		if a[i] == b {
			return false
		}
	}
	return true
}

func createTemporaryFile() (string, string, *os.File, error) {
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
	p := make([]byte, DefaultJunkFileSizeInMB*1024*1024)
	_, err := rand.Read(p)
	if err != nil {
		return err
	}

	_, err = file.Write(p)
	return err
}

func getChecksumForBlocks(blocks []Block) (string, error) {
	blockSizeInBytes := BlockSizeInKB * 1024
	p := make([]byte, len(blocks)*blockSizeInBytes)
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

	e = Engine{
		db:              testDB,
		blockSizeInKB:   BlockSizeInKB,
		storageLocation: os.TempDir(),
	}

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
