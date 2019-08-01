/*
This file is part of CProxy-Cache.

CProxy-Cache is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

CProxy-Cache is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with CProxy-Cache.  If not, see <https://www.gnu.org/licenses/>.
*/

package ccache

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"net/http"
	"os"
	"strings"
)

// CacheStorageFile - storage handler name for file system handler
const CacheStorageFile = "file"

// CacheStorageMemory - storage handler name for memory handler
const CacheStorageMemory = "memory"

// CacheItemFileExtension - file extension to use for
const CacheItemFileExtension = ".ccache"

// Storage - define storage handler methods
type Storage interface {
	Init(key string, config *Config)
	GetTypeName() string
	StoreResponse(r *http.Response) error
	FetchResponse() (*http.Response, error)
	GetSize() (int64, error)
	Delete() error
}

// FileStorage - file system storage handler
type FileStorage struct {
	config *Config
	key    string
}

// Init - init storage handler
func (s *FileStorage) Init(key string, config *Config) {
	s.key = key
	s.config = config
}

// GetTypeName - get storage handler name
func (s *FileStorage) GetTypeName() string {
	return CacheStorageFile
}

// getFilePath - get path to cache file
func (s *FileStorage) getFilePath() (string, error) {
	if s.key == "" {
		return "", errors.New("cannot use file system storage without cache key")
	}
	if s.config == nil {
		return "", errors.New("cannot use file system storage without config")
	}
	cacheFilePath := strings.TrimRight(s.config.CacheFilePath, "/") + "/"
	return cacheFilePath + s.key + CacheItemFileExtension, nil
}

// StoreResponse - store http response to file system
func (s *FileStorage) StoreResponse(r *http.Response) error {
	// get file path
	cacheFileName, err := s.getFilePath()
	if err != nil {
		return err
	}
	// store cache item in tmp
	f, err := os.Create(cacheFileName)
	if err != nil {
		return err
	}
	defer f.Close()
	gw, err := gzip.NewWriterLevel(f, gzip.BestSpeed)
	if err != nil {
		return err
	}
	defer gw.Close()
	return r.Write(gw)
}

// FetchResponse - fetch response from file system
func (s *FileStorage) FetchResponse() (*http.Response, error) {
	// get file path
	cacheFileName, err := s.getFilePath()
	if err != nil {
		return nil, err
	}
	// open cached response, ungzip
	f, err := os.Open(cacheFileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	// read response buf in to memory
	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(gr)
	if err != nil {
		return nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(buf), nil)
	return resp, err
}

// GetSize - get file size of cache item
func (s *FileStorage) GetSize() (int64, error) {
	// get file path
	cacheFileName, err := s.getFilePath()
	if err != nil {
		return 0, err
	}
	// open file
	f, err := os.Open(cacheFileName)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	// state file
	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

// Delete - delete file
func (s *FileStorage) Delete() error {
	// get file path
	cacheFileName, err := s.getFilePath()
	if err != nil {
		return err
	}
	return os.Remove(cacheFileName)
}

// MemoryStorage - memory storage handler
type MemoryStorage struct {
	config *Config
	key    string
	data   []byte
}

// Init - init storage handler
func (s *MemoryStorage) Init(key string, config *Config) {
	s.key = key
	s.config = config
	s.data = make([]byte, 0)
}

// GetTypeName - get storage handler name
func (s *MemoryStorage) GetTypeName() string {
	return CacheStorageMemory
}

// StoreResponse - store http response to file system
func (s *MemoryStorage) StoreResponse(r *http.Response) error {
	buf := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(buf)
	defer gw.Close()
	err := r.Write(gw)
	if err != nil {
		return err
	}
	gw.Close()
	s.data = make([]byte, buf.Len())
	_, err = buf.Read(s.data)
	return err
}

// FetchResponse - fetch response from file system
func (s *MemoryStorage) FetchResponse() (*http.Response, error) {
	// read from memory, uncompress
	br := bytes.NewReader(s.data)
	gr, err := gzip.NewReader(br)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	resp, err := http.ReadResponse(bufio.NewReader(gr), nil)
	return resp, err
}

// GetSize - get file size of cache item
func (s *MemoryStorage) GetSize() (int64, error) {
	return int64(len(s.data)), nil
}

// Delete - delete file
func (s *MemoryStorage) Delete() error {
	s.data = make([]byte, 0)
	return nil
}

// GetStorageHandler - get a storage handler from its name
func GetStorageHandler(name string) Storage {
	switch name {
	case CacheStorageFile:
		{
			return &FileStorage{}
		}
	case CacheStorageMemory:
		{
			return &MemoryStorage{}
		}
	}
	return nil
}
