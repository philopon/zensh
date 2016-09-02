package util

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ulikunitz/xz"
)

type DirectoryAlreadyExists string

func (dae DirectoryAlreadyExists) Error() string {
	return fmt.Sprintf("directory already exists: %v", string(dae))
}

type UnknownTarTypeFlag byte

func (uttf UnknownTarTypeFlag) Error() string {
	return fmt.Sprintf("unknown tar typeflag: %v", rune(uttf))
}

type PeekableReader interface {
	io.Reader
	Peekable
}

func Unarchive(dst, defName string, file PeekableReader) error {
	file_type, err := Detect(file)
	if err != nil {
		return err
	}

	switch file_type {
	case GZip:
		gz, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gz.Close()
		return Unarchive(dst, defName, bufio.NewReader(gz))

	case BZip:
		return Unarchive(dst, defName, bufio.NewReader(bzip2.NewReader(file)))

	case Xz:
		xz, err := xz.NewReader(file)
		if err != nil {
			return err
		}
		return Unarchive(dst, defName, bufio.NewReader(xz))

	case Tar:
		if err := unarchiveTar(dst, tar.NewReader(file)); err != nil {
			return err
		}
		return undirectory(dst)

	case Zip:
		tmp, err := ioutil.TempFile("", "temp")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		writer := bufio.NewWriter(tmp)

		if _, err := writer.ReadFrom(file); err != nil {
			return err
		}

		if err := writer.Flush(); err != nil {
			return err
		}

		if err := tmp.Close(); err != nil {
			return err
		}

		rdr, err := zip.OpenReader(tmp.Name())
		if err != nil {
			return err
		}
		defer rdr.Close()

		if err := unarchiveZip(dst, rdr); err != nil {
			return err
		}
		return undirectory(dst)

	default:
		if err := unarchiveMkbase(dst); err != nil {
			return err
		}

		name := filepath.Join(dst, defName)
		out, err := os.OpenFile(name, os.O_CREATE+os.O_WRONLY, 0755)
		if err != nil {
			return err
		}
		defer out.Close()

		writer := bufio.NewWriter(out)

		if _, err := writer.ReadFrom(file); err != nil {
			return err
		}

		if err := writer.Flush(); err != nil {
			return err
		}

		return nil
	}
}

func unarchiveMkbase(dir string) error {
	if stat, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if !stat.IsDir() {
		return DirectoryAlreadyExists(dir)
	}

	return nil
}

func unarchiveTar(dst string, archive *tar.Reader) error {
	for {
		hdr, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		name := filepath.Join(dst, hdr.Name)
		info := hdr.FileInfo()
		dir := filepath.Dir(name)

		if err := unarchiveMkbase(dir); err != nil {
			return err
		}

		switch flag := hdr.Typeflag; flag {
		case tar.TypeReg, tar.TypeRegA:
			out, err := os.OpenFile(name, os.O_WRONLY+os.O_CREATE, info.Mode())
			if err != nil {
				return err
			}
			defer out.Close()

			writer := bufio.NewWriter(out)
			if _, err := writer.ReadFrom(archive); err != nil {
				return err
			}

			if err := writer.Flush(); err != nil {
				return err
			}

		case tar.TypeLink:
			if err := os.Link(hdr.Linkname, name); err != nil {
				return err
			}

		case tar.TypeSymlink:
			if err := os.Symlink(hdr.Linkname, name); err != nil {
				return err
			}

		case tar.TypeDir:
			if err := os.MkdirAll(name, info.Mode()); err != nil {
				return err
			}

		default:
			return UnknownTarTypeFlag(flag)
		}

	}

	return nil
}

func unarchiveZip(dst string, archive *zip.ReadCloser) error {
	for _, file := range archive.File {
		name := filepath.Join(dst, file.Name)
		info := file.FileInfo()
		dir := filepath.Dir(name)

		if err := unarchiveMkbase(dir); err != nil {
			return err
		}

		if info.IsDir() {
			if err := os.MkdirAll(name, info.Mode()); err != nil {
				return err
			}
		} else {
			out, err := os.OpenFile(name, os.O_WRONLY+os.O_CREATE, info.Mode())
			if err != nil {
				return err
			}
			defer out.Close()

			writer := bufio.NewWriter(out)

			zf, err := file.Open()
			if err != nil {
				return err
			}
			defer zf.Close()

			if _, err := writer.ReadFrom(zf); err != nil {
				return err
			}

			if err := writer.Flush(); err != nil {
				return err
			}
		}
	}
	return nil
}

func undirectory(dst string) error {
	files, err := ioutil.ReadDir(dst)
	if err != nil {
		return err
	}
	if len(files) == 1 && files[0].IsDir() {
		childInfo := files[0]
		child := filepath.Join(dst, childInfo.Name())

		grandchildren, err := ioutil.ReadDir(child)
		if err != nil {
			return err
		}

		for _, grandchildInfo := range grandchildren {
			grandchild := filepath.Join(child, grandchildInfo.Name())
			to := filepath.Join(dst, grandchildInfo.Name())
			os.Rename(grandchild, to)
		}

		os.Remove(child)
		undirectory(dst)
	}

	return nil
}
