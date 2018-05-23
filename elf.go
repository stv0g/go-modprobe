package modprobe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"debug/elf"

	"golang.org/x/sys/unix"
)

func modulePath(path string) (string, error) {
	uname := unix.Utsname{}
	if err := unix.Uname(&uname); err != nil {
		return "", err
	}

	i := 0
	for ; uname.Release[i] != 0; i++ {
	}

	return filepath.Join(
		"/lib/modules",
		string(uname.Release[:i]),
		path,
	), nil
}

func resolveName(name string) (string, error) {
	paths, err := generateMap()
	if err != nil {
		return "", err
	}
	root, err := modulePath("")
	if err != nil {
		return "", err
	}

	fsPath := paths[name]
	if !strings.HasPrefix(fsPath, root) {
		return "", fmt.Errorf("Module isn't in the module directory")
	}

	relPath := fsPath[len(root)+1:]
	return relPath, nil
}

// Open every single kernel module under the kernel module directory
// (/lib/modules/$(uname -r)/), and parse the ELF headers to extract the
// module name.
func generateMap() (map[string]string, error) {
	mPath, err := modulePath("")
	if err != nil {
		return nil, err
	}
	return elfMap(mPath)
}

// Open every single kernel module under the root, and parse the ELF headers to
// extract the module name.
func elfMap(root string) (map[string]string, error) {
	ret := map[string]string{}

	err := filepath.Walk(
		root,
		func(path string, info os.FileInfo, err error) error {
			if !info.Mode().IsRegular() {
				return nil
			}
			fd, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fd.Close()
			name, err := Name(fd)
			if err != nil {
				/* For now, let's just ignore that and avoid adding to it */
				return nil
			}

			ret[name] = path
			return nil
		})

	if err != nil {
		return nil, err
	}

	return ret, nil
}

// Given a file descriptor, go ahead and parse out the module name from the
// Symbols.
func Name(file *os.File) (string, error) {
	f, err := elf.NewFile(file)
	if err != nil {
		return "", err
	}

	syms, err := f.Symbols()
	if err != nil {
		return "", err
	}

	for _, sym := range syms {
		if strings.Compare(sym.Name, "__this_module") == 0 {
			section := f.Sections[sym.Section]
			data, err := section.Data()
			if err != nil {
				return "", err
			}

			data = data[24:]
			i := 0
			for ; data[i] != 0x00; i++ {
			}
			return string(data[:i]), nil
		}
	}

	return "", nil
	// .gnu.linkonce.this_module
}
