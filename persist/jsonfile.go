package persist

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
)

type JSONFile struct {
	file           *os.File
	fileNameFormat string
	splitsize      int64
	mu             sync.Mutex
}

func newJSONFile(o *PersistOptions) (*JSONFile, error) {
	file, err := os.OpenFile(o.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, o.fileMode)
	return &JSONFile{
		file:           file,
		splitsize:      o.fileSize,
		fileNameFormat: o.flieNameFormat,
		mu:             sync.Mutex{},
	}, err
}

func (this *JSONFile) Save(item interface{}) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	this.mu.Lock()
	defer this.mu.Unlock()

	fileInfo, err := this.file.Stat()
	if err != nil {
		return err
	}

	if fileInfo.Size() >= this.splitsize {
		if err = this.splitFile(); err != nil {
			panic(err.Error())
		}
	}
	_, err = this.file.Write(data)
	return err
}
func (this *JSONFile) Close() {
	dir, fileName := path.Split(this.file.Name())

	f := strings.Split(fileName, ".")

	var newPath string

	for i := 1; ; i++ {
		newPath = fmt.Sprintf("%s"+this.fileNameFormat, dir, i, f[0], f[1])
		if _, err := os.Stat(newPath); err != nil && os.IsNotExist(err) {

			break
		}
	}
	os.Rename(this.file.Name(), newPath)

	this.file.Close()
}
func (this *JSONFile) splitFile() (err error) {
	defer this.file.Close()

	dir, fileName := path.Split(this.file.Name())

	f := strings.Split(fileName, ".")

	var newPath string

	for i := 1; ; i++ {
		newPath = fmt.Sprintf("%s"+this.fileNameFormat, dir, i, f[0], f[1])
		if _, err := os.Stat(newPath); err != nil && os.IsNotExist(err) {
			break
		}
	}
	if err = os.Rename(this.file.Name(), newPath); err != nil {
		return
	}

	this.file, err = os.OpenFile(this.file.Name(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	return
}
