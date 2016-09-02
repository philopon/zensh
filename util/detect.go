package util

const detectLength = 0x101 + 8

type Filetype int

const (
	Unknown Filetype = iota
	GZip
	BZip
	Xz
	Zip
	Tar
)

func (t Filetype) String() string {
	switch t {
	case GZip:
		return "GZip"
	case BZip:
		return "BZip"
	case Xz:
		return "Xz"
	case Zip:
		return "Zip"
	case Tar:
		return "Tar"
	default:
		return "Unknown"
	}
}

func match(str []byte, desired []byte, offset int) bool {
	if len(str) < len(desired)+offset {
		return false
	}

	str = str[offset:]
	for i := 0; i < len(desired); i++ {
		if str[i] == desired[i] {
			continue
		}

		return false
	}

	return true
}

type Peekable interface {
	Peek(n int) ([]byte, error)
}

func Detect(rdr Peekable) (Filetype, error) {
	head, err := rdr.Peek(detectLength)
	if err != nil {
		return Unknown, err
	}

	switch {
	case match(head, []byte("\x1f\x8b"), 0):
		return GZip, nil
	case match(head, []byte("BZh"), 0):
		return BZip, nil
	case match(head, []byte("\xfd7zXZ\x00"), 0):
		return Xz, nil
	case match(head, []byte("PK\x03\x04"), 0):
		return Zip, nil
	case match(head, []byte("ustar\x0000"), 0x101):
		return Tar, nil
	case match(head, []byte("ustar  \x00"), 0x101):
		return Tar, nil
	}

	return Unknown, nil
}
