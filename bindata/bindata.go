// Code generated for package bindata by go-bindata DO NOT EDIT. (@generated)
// sources:
// the-hook/Dockerfile
// the-hook/deployment-patch.yaml
package bindata

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _theHookDockerfile = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x24\xc9\xb1\x8a\x83\x30\x1c\x07\xe0\x3d\x4f\xf1\xbb\x70\xe3\x79\x11\x0e\x6e\xa8\x74\x70\xb0\xe2\xa0\x96\x60\x87\x52\x3a\x68\xf2\x27\x11\x12\x53\x6a\xb4\xf4\xed\x8b\xed\xf2\x2d\x5f\x2e\x4b\x54\x75\x5e\x16\x7b\x13\x5c\x3f\x99\x9d\xeb\x23\xcd\x91\x1d\x64\x5b\xe3\xfb\x3d\x8c\xc9\x53\x03\x13\x60\x28\xc2\x8c\xd1\x2e\xc3\xaf\x0a\x5e\x98\x90\x68\x72\x2b\x89\x8f\xca\x6b\xa1\xdd\xca\x58\xd1\x74\xf2\x7c\x6c\xab\xa6\xc3\x85\x8b\x61\x9c\xc4\x6c\xf9\x0f\x78\xa2\x36\x1f\x76\x74\x84\x78\x5f\x28\x83\x0e\x20\x65\x03\xf4\xe2\xfd\xf3\x2b\xc3\xec\x88\x6e\xf8\xfb\x4f\xd3\xed\x26\xe2\xd7\x57\x00\x00\x00\xff\xff\xdc\x9d\x89\x8e\x9f\x00\x00\x00")

func theHookDockerfileBytes() ([]byte, error) {
	return bindataRead(
		_theHookDockerfile,
		"the-hook/Dockerfile",
	)
}

func theHookDockerfile() (*asset, error) {
	bytes, err := theHookDockerfileBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "the-hook/Dockerfile", size: 159, mode: os.FileMode(420), modTime: time.Unix(1593100633, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _theHookDeploymentPatchYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x51\xcb\x6e\x83\x30\x10\xbc\xe7\x2b\x46\x28\xc7\x92\x0f\x40\xea\xa9\x97\x56\x6a\x2b\x7e\x61\x83\xb7\x60\xc9\x0f\x64\x9b\x5c\x10\xff\x5e\xd9\x86\x00\x51\xaa\xc2\xc9\x3b\x33\x3b\xbb\xb3\x9c\x7c\xcf\x4d\x75\x02\x02\xeb\x5e\x51\xe0\xf8\x06\x34\x07\x12\x14\x28\x57\x80\xa2\x2b\x2b\xbf\x54\xc0\x38\xe2\x52\x53\x68\x3a\x16\x9f\x91\xfa\x26\xcd\x98\xa6\x0a\x45\x70\x03\x17\x49\xb7\x38\xc7\xaf\xb1\x26\x90\x34\xec\xee\x1e\x25\x0c\x69\xae\x92\xd3\xdb\xc2\xce\x36\xf7\x31\x52\x53\x3b\x6b\x3e\xe2\x73\xcb\x35\x56\x6b\x32\x62\xdd\x89\x5c\xbb\xd9\x50\xc9\x1b\x1b\xf6\xbe\x76\xf6\xca\x2b\xec\x98\x84\x7c\x82\x8f\x23\xe4\x0f\x2e\x5f\x76\x30\xc1\x6f\xc7\xdc\xac\x1a\x34\x67\x7c\x95\xa7\x06\x47\xa6\x65\x9c\xe5\x0b\xce\x3a\xf2\xa8\x5e\x9f\x39\xc4\xac\x89\xaf\x29\x74\x29\x4c\x96\x67\x69\x04\xf7\x6a\xcc\x97\x29\xfa\x78\xe0\x32\x69\xcb\xd8\x25\x31\x4d\xc5\x7e\x05\x36\x62\xdb\xfc\x88\xfc\x91\x2a\x67\xda\xff\xce\x83\x61\xca\xa3\xcb\x75\xd6\xe7\xc0\xbb\x64\xfd\xc3\x09\xde\x67\xd5\x3f\x21\x72\xfd\x1b\x00\x00\xff\xff\x94\x5a\x0c\x9d\xac\x02\x00\x00")

func theHookDeploymentPatchYamlBytes() ([]byte, error) {
	return bindataRead(
		_theHookDeploymentPatchYaml,
		"the-hook/deployment-patch.yaml",
	)
}

func theHookDeploymentPatchYaml() (*asset, error) {
	bytes, err := theHookDeploymentPatchYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "the-hook/deployment-patch.yaml", size: 684, mode: os.FileMode(420), modTime: time.Unix(1593100633, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"the-hook/Dockerfile":            theHookDockerfile,
	"the-hook/deployment-patch.yaml": theHookDeploymentPatchYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"the-hook": &bintree{nil, map[string]*bintree{
		"Dockerfile":            &bintree{theHookDockerfile, map[string]*bintree{}},
		"deployment-patch.yaml": &bintree{theHookDeploymentPatchYaml, map[string]*bintree{}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
