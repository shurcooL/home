package fs

import (
	"context"
	"encoding/json"
	"io"
	"os"
	pathpkg "path"

	"github.com/shurcooL/webdavfs/vfsutil"
	"golang.org/x/net/webdav"
)

// jsonAppendFile encodes v into file at path, appending to or creating it.
// The parent directory must exist, otherwise an error will be returned.
func jsonAppendFile(ctx context.Context, fs webdav.FileSystem, path string, v interface{}) error {
	f, err := fs.OpenFile(ctx, path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// jsonDecodeAllFile decodes all contents of file at path into vs.
func jsonDecodeAllFile(ctx context.Context, fs webdav.FileSystem, path string, vs *[]notificationDisk) error {
	f, err := vfsutil.Open(ctx, fs, path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	for {
		var v notificationDisk
		err := dec.Decode(&v)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		*vs = append(*vs, v)
	}
	return nil
}

// createEmptyFile creates an empty file at path, creating parent directories if needed.
func createEmptyFile(ctx context.Context, fs webdav.FileSystem, path string) error {
	f, err := vfsutil.Create(ctx, fs, path)
	if os.IsNotExist(err) {
		err = vfsutil.MkdirAll(ctx, fs, pathpkg.Dir(path), 0755)
		if err != nil {
			return err
		}
		f, err = vfsutil.Create(ctx, fs, path)
	}
	if err != nil {
		return err
	}
	_ = f.Close()
	return nil
}

// appendFile appends the contents of file at src to the end of file at dst.
func appendFile(ctx context.Context, fs webdav.FileSystem, dst, src string) error {
	s, err := vfsutil.Open(ctx, fs, src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := fs.OpenFile(ctx, dst, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer d.Close()
	_, err = io.Copy(d, s)
	return err
}
