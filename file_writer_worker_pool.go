package edis

import (
	"fmt"
	"os"
	"sync"

	"github.com/ncw/directio"

	"github.com/spacemonkeygo/openssl"
)

type blockWriteTask struct {
	blockNumber int
	buffer      []byte
}

type blockWriteResult struct {
	path        string
	isNew       bool
	blockNumber int
	checksum    string
}

type fileWriterWorkerPool struct {
	bufferSize        int
	e                 *Engine
	ov                ObjectVersion
	file              *os.File
	writer            chan blockWriteTask
	filler            chan []byte
	finished          chan blockWriteResult
	isDirectIOEnabled bool
}

func makeFileWriterWorkerPool(e *Engine, ov ObjectVersion,
	f *os.File, isDirectIOEnabled bool) *fileWriterWorkerPool {
	return &fileWriterWorkerPool{
		bufferSize:        ov.BlockSize,
		e:                 e,
		ov:                ov,
		file:              f,
		filler:            make(chan []byte),
		finished:          make(chan blockWriteResult),
		writer:            make(chan blockWriteTask),
		isDirectIOEnabled: isDirectIOEnabled,
	}
}

func (wp *fileWriterWorkerPool) write() ([]blockWriteResult, error) {
	if err := wp.start(); err != nil {
		return []blockWriteResult{}, err
	}
	return wp.getResults(), nil
}

func (wp *fileWriterWorkerPool) start() error {
	if err := wp.startAsynchronousWriter(); err != nil {
		return err
	}

	if err := wp.startAsynchronousReader(); err != nil {
		return err
	}

	bufferA := wp.makeBufferForFile()
	bufferB := wp.makeBufferForFile()
	wp.filler <- bufferA
	if wp.ov.NumberOfBlocks > 1 {
		wp.filler <- bufferB
	}

	return nil
}

func (wp *fileWriterWorkerPool) makeBufferForFile() []byte {
	if wp.isDirectIOEnabled {
		return directio.AlignedBlock(wp.bufferSize)
	}
	return make([]byte, wp.bufferSize)
}

func (wp *fileWriterWorkerPool) getResults() []blockWriteResult {
	wg := sync.WaitGroup{}
	wg.Add(wp.ov.NumberOfBlocks)

	var wr []blockWriteResult
	go func() {
		for len(wr) < wp.ov.NumberOfBlocks {
			x := <-wp.finished
			wr = append(wr, x)
			wg.Done()
		}
	}()

	wg.Wait()
	return wr
}

func (wp *fileWriterWorkerPool) startAsynchronousReader() error {
	_, err := wp.file.Seek(0, 0)
	if err != nil {
		return err
	}

	go func() {
		blockNumber := 0
		for blockNumber < wp.ov.NumberOfBlocks {
			buffer := <-wp.filler
			isBlockDefinitelyFull := blockNumber < wp.ov.NumberOfBlocks-1
			if isBlockDefinitelyFull {
				_, err = wp.file.Read(buffer)
			} else {
				buffer, err = wp.e.getBlockInFile(wp.file, wp.ov.BlockSize, blockNumber)
			}

			if err != nil {
				panic(err)
			}

			wp.writer <- blockWriteTask{blockNumber, buffer}
			blockNumber++
		}
	}()

	return nil
}

func (wp *fileWriterWorkerPool) startAsynchronousWriter() error {
	go func() {
		for true {
			task := <-wp.writer
			hash, err := openssl.SHA1(task.buffer)
			if err != nil {
				panic(err)
			}

			blockChecksum := fmt.Sprintf("%x", hash)
			isBlockNew, err := wp.e.isBlockNew(wp.ov, task.blockNumber, blockChecksum)
			if err != nil {
				panic(err)
			}

			if isBlockNew {
				var pathToBlock string
				isFileContentNew, err := wp.e.isFileContentNew(blockChecksum)
				if !isFileContentNew {
					pathToBlock, err = wp.e.writeBytesAsBlock(wp.ov, task.blockNumber, task.buffer)
					if err != nil {
						panic(err)
					}
				} else {
					pathToBlock, err = wp.e.getPathForBlockWithChecksum(blockChecksum)
					if err != nil {
						panic(err)
					}
				}

				go func() {
					wp.finished <- blockWriteResult{pathToBlock, true, task.blockNumber, blockChecksum}
				}()
			} else {
				go func() {
					wp.finished <- blockWriteResult{"", false, task.blockNumber, blockChecksum}
				}()
			}

			if task.blockNumber == wp.ov.NumberOfBlocks-1 {
				break
			} else {
				go func() {
					wp.filler <- task.buffer
				}()
			}
		}
	}()

	return nil
}
