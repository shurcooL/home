package fs

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strconv"

	"github.com/shurcooL/webdavfs/vfsutil"
	"golang.org/x/net/webdav"
)

// fileInfoID describes a file, whose name is an ID of type uint64.
type fileInfoID struct {
	os.FileInfo
	ID uint64
}

// byID implements sort.Interface.
type byID []fileInfoID

func (f byID) Len() int           { return len(f) }
func (f byID) Less(i, j int) bool { return f[i].ID < f[j].ID }
func (f byID) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }

// readDirIDs reads the directory named by path and returns
// a list of directory entries whose names are IDs of type uint64, sorted by ID.
// Other entries with names don't match the naming scheme are ignored.
// If the directory doesn't exist, a not exist error is returned.
func readDirIDs(ctx context.Context, fs webdav.FileSystem, path string) ([]fileInfoID, error) {
	fis, err := vfsutil.ReadDir(ctx, fs, path)
	if err != nil {
		return nil, err
	}
	var fiis []fileInfoID
	for _, fi := range fis {
		id, err := strconv.ParseUint(fi.Name(), 10, 64)
		if err != nil {
			continue
		}
		fiis = append(fiis, fileInfoID{
			FileInfo: fi,
			ID:       id,
		})
	}
	sort.Sort(byID(fiis))
	return fiis, nil
}

// jsonEncodeFile encodes v into file at path, overwriting or creating it.
func jsonEncodeFile(ctx context.Context, fs webdav.FileSystem, path string, v interface{}) error {
	f, err := fs.OpenFile(ctx, path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(v)
}

// jsonDecodeFile decodes contents of file at path into v.
func jsonDecodeFile(ctx context.Context, fs webdav.FileSystem, path string, v interface{}) error {
	f, err := vfsutil.Open(ctx, fs, path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}
