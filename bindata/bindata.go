// Code generated by go-bindata. DO NOT EDIT.
// sources:
// the-hook/Dockerfile (167B)
// the-hook/deployment-patch.yaml (755B)

package bindata

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
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
		return nil, fmt.Errorf("read %q: %w", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %w", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes  []byte
	info   os.FileInfo
	digest [sha256.Size]byte
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _theHookDockerfile = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x24\xca\xb1\x6e\x83\x30\x10\x80\xe1\xdd\x4f\x71\xb5\x3a\xd6\x35\x52\xa5\x0e\x45\x1d\x18\x08\x62\x00\x22\x8b\x0c\x51\x94\x01\xec\xd3\x19\xc9\x70\x08\x0c\x79\xfd\x88\x64\xf9\x96\xff\xcf\x4c\x01\x65\x95\x15\xf9\x3f\x71\xe8\x26\xfa\x0b\x5d\xc4\x35\x8a\x93\x69\x2a\xf8\x7c\x15\x21\xcc\xa5\x06\x62\x20\x8c\x40\x43\xf4\x5b\xff\x6d\x79\xd4\xc4\xca\x61\xd8\x51\xbf\xb5\xa3\xd3\x2e\xec\x42\xe4\x75\x6b\xae\xe7\xa6\xac\x5b\xb8\x49\xdd\x0f\x93\x5e\xbd\xfc\x02\xa9\xec\xe1\xc3\x0f\x01\x21\x2e\x1b\xa6\xe0\x18\xd0\x7a\x06\x62\x45\x4b\x37\xcf\x01\x15\xf1\x47\x0a\x6b\x40\x9c\xe1\xe7\x37\x49\x8e\x67\x42\x79\x7f\x06\x00\x00\xff\xff\xee\xc6\x23\xf3\xa7\x00\x00\x00")

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

	info := bindataFileInfo{name: "the-hook/Dockerfile", size: 167, mode: os.FileMode(0644), modTime: time.Unix(1593158731, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x18, 0x2, 0xa4, 0x80, 0xee, 0xd7, 0xf2, 0x95, 0x86, 0x3b, 0x54, 0xd8, 0xe4, 0x7a, 0x6b, 0x60, 0xee, 0x69, 0x95, 0x29, 0xe2, 0x7a, 0xed, 0x33, 0x25, 0x75, 0x8, 0xa, 0x62, 0xf5, 0x6, 0xe4}}
	return a, nil
}

var _theHookDeploymentPatchYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x91\x41\x6a\xf3\x30\x10\x85\xf7\x39\xc5\x60\xbc\xfc\x9d\x03\x18\xfe\x55\xbb\x68\xa1\x81\x5c\x61\x22\x4f\x6c\x81\x46\x12\xb2\x1c\x28\x46\x77\x2f\x92\x62\x5b\x4e\x4b\x1b\xaf\xa4\x37\x33\x4f\xef\x1b\x1f\x46\x4b\xa2\x3d\x00\x78\x62\xab\xd0\x53\x3c\x03\x30\x79\xec\xd0\x63\xbe\x01\x28\xbc\x90\x1a\x97\x1b\xc0\x3c\xc3\xf1\x23\x6a\x10\x42\x0b\x95\x77\x13\x55\xa9\xb8\xd8\xc5\x4f\x18\xed\x51\x6a\x72\xeb\x60\x03\x1a\x99\xda\x34\xfe\xb2\x54\x21\x84\xd5\x57\x32\xf6\xf7\xfa\x7b\x3c\x96\x35\x61\x98\x51\x77\x5b\x08\x74\x7d\x11\x49\xc9\x1b\x69\x1a\xc7\xb3\x33\x17\xda\x64\x47\xd8\xc9\x1f\xf4\x9b\x51\x13\xd3\xc9\x4c\xda\x17\x26\x5b\x44\x8b\x5e\x0c\x8d\x30\xfa\x2a\x7b\x46\x5b\x74\x00\x70\x9c\x3a\xa3\x1f\x56\x92\xab\xec\x4f\x68\x93\x5b\x19\x39\x2d\xca\xa1\xee\x09\x6a\xf9\x0f\xea\x34\x08\xed\x7f\x38\xe6\x87\xf7\xbd\xcb\xd3\x55\x7e\x3b\x35\x37\xf3\x0c\xb5\x84\x10\xaa\x5f\x12\x64\xdf\xec\x19\xc5\x6f\x11\x48\x77\x9b\x96\xc9\x0b\xe8\xbf\x91\xc5\x42\xd8\xee\x52\x6c\x3f\xf3\x95\xac\x32\x9f\x4c\x7b\xfc\xe7\xe1\x9f\x46\x1f\xcc\x98\xb9\x77\x41\xec\xc3\x26\xde\xee\x5d\x0f\x61\xf2\x1a\xbe\x02\x00\x00\xff\xff\x97\x35\x00\xfe\xf3\x02\x00\x00")

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

	info := bindataFileInfo{name: "the-hook/deployment-patch.yaml", size: 755, mode: os.FileMode(0644), modTime: time.Unix(1593534251, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xf, 0xe6, 0x33, 0x57, 0xf2, 0x12, 0x93, 0x8d, 0xdb, 0xaf, 0x57, 0xad, 0x4f, 0x8d, 0x5, 0x64, 0xbb, 0xc1, 0x63, 0xb9, 0x8d, 0xee, 0xba, 0xc7, 0x60, 0xd4, 0x53, 0xd, 0x8b, 0xe9, 0x16, 0xc2}}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetString returns the asset contents as a string (instead of a []byte).
func AssetString(name string) (string, error) {
	data, err := Asset(name)
	return string(data), err
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

// MustAssetString is like AssetString but panics when Asset would return an
// error. It simplifies safe initialization of global variables.
func MustAssetString(name string) string {
	return string(MustAsset(name))
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetDigest returns the digest of the file with the given name. It returns an
// error if the asset could not be found or the digest could not be loaded.
func AssetDigest(name string) ([sha256.Size]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s can't read by error: %v", name, err)
		}
		return a.digest, nil
	}
	return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s not found", name)
}

// Digests returns a map of all known files and their checksums.
func Digests() (map[string][sha256.Size]byte, error) {
	mp := make(map[string][sha256.Size]byte, len(_bindata))
	for name := range _bindata {
		a, err := _bindata[name]()
		if err != nil {
			return nil, err
		}
		mp[name] = a.digest
	}
	return mp, nil
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

// AssetDebug is true if the assets were built with the debug flag enabled.
const AssetDebug = false

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"},
// AssetDir("data/img") would return []string{"a.png", "b.png"},
// AssetDir("foo.txt") and AssetDir("notexist") would return an error, and
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		canonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(canonicalName, "/")
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

// RestoreAsset restores an asset under the given directory.
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
	return os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
}

// RestoreAssets restores an asset under the given directory recursively.
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
	canonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(canonicalName, "/")...)...)
}
