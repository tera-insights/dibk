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
	objectName, path, file, err := createTemporaryFile()
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

func TestChangingBlocksWithSameSizeFile(t *testing.T) {
	e := Engine{
		db:              testDB,
		blockSizeInKB:   BlockSizeInKB,
		storageLocation: os.TempDir(),
	}

	objectName, path, file, err := createTemporaryFile()
	if err != nil {
		t.Fatal(err)
	}

	err = writeToJunkFile(file)
	if err != nil {
		t.Fatal(err)
	}

	err = e.saveObject(file, objectName, 1)
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

	fmt.Printf("nBlocks = %d\n", nBlocks)

	newBytes := make([]byte, nBlocks*BlockSizeInKB*1024)
	oldBytes := make([]byte, nBlocks*BlockSizeInKB*1024)
	file, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	_, err = file.Read(oldBytes)
	if err != nil {
		t.Fatal(err)
	}

	copy(newBytes, oldBytes)

	nToChange := 2
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

	_, newPath, newFile, err := createTemporaryFile()
	if err != nil {
		t.Fatal(err)
	}

	_, err = newFile.Write(newBytes)
	if err != nil {
		t.Fatal(err)
	}

	err = e.saveObject(newFile, objectName, 2)
	if err != nil {
		t.Fatal(err)
	}

	fetchedVersionOneBlocks, err := e.loadBlockInfos(objectName, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(correctVersionOneBlocks) != len(fetchedVersionOneBlocks) {
		t.Fatalf("Lengths of blocks differ")
	}

	for i := 0; i < len(correctVersionOneBlocks); i++ {
		correct := correctVersionOneBlocks[i]
		fetched := fetchedVersionOneBlocks[i]
		isCorrect := (correct.BlockIndex == fetched.BlockIndex &&
			correct.Location == fetched.Location &&
			correct.ObjectName == fetched.ObjectName &&
			correct.SHA256Checksum == fetched.SHA256Checksum &&
			correct.Version == fetched.Version)
		if !isCorrect {
			t.Fatalf("Original version one blocks did not equal those we just fetched")
		}
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
		for j := 0; j < len(changedIndices); j++ {
			isCorrectVersion := (j == i && block.Version == 2) ||
				(j != i && block.Version == 1)
			if !isCorrectVersion {
				t.Fatalf("Block versions did not match what was changed")
			}
		}
	}

	fileChecksum, err := getChecksumForPath(newPath, JunkFileSizeInMB*1024*1024)
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
