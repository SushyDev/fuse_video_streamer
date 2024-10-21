package fuse

import (
	"debrid_drive/vfs"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

var _ fs.FS = &FuseFileSystem{}

type FuseFileSystem struct {
	vfs        *vfs.FileSystem
	connection *fuse.Conn
}

func NewFileSystem(connection *fuse.Conn, vfs *vfs.FileSystem) *FuseFileSystem {
	fuseFileSystem := &FuseFileSystem{
		connection: connection,
		vfs:        vfs,
	}

	return fuseFileSystem
}

func (fileSystem *FuseFileSystem) Root() (fs.Node, error) {
	root := NewDirectoryNode(fileSystem.connection, fileSystem.vfs.Root)

	return root, nil
}

// func (fileSystem *FuseFileSystem) AddDirectory(parent *Directory, name string) (*Directory, error) {
// 	if parent == nil {
// 		return nil, fmt.Errorf("parent directory is nil")
// 	}
//
// 	directoryFound, _ := parent.GetDirectory(name)
// 	if directoryFound != nil {
// 		return nil, fmt.Errorf("directory %s already exists", name)
// 	}
//
// 	newDirectory := fileSystem.NewDirectory(parent, name)
//
// 	fileSystem.mu.Lock()
// 	parent.directories[newDirectory.GetINode()] = newDirectory
// 	fileSystem.directoryMap[newDirectory.iNode] = newDirectory
// 	fileSystem.mu.Unlock()
//
// 	defer parent.Invalidate()
//
// 	return newDirectory, nil
// }

// func (fileSystem *FuseFileSystem) RemoveDirectory(parent *Directory, nodeId uint64) {
// 	fileSystem.mu.Lock()
// 	defer fileSystem.mu.Unlock()
//
// 	delete(parent.directories, nodeId)
// 	delete(fileSystem.directoryMap, parent.iNode)
//
// 	defer parent.Invalidate()
// }

// func (fileSystem *FuseFileSystem) GetDirectory(iNode uint64) (*Directory, error) {
// 	fileSystem.mu.RLock()
// 	defer fileSystem.mu.RUnlock()
//
// 	if iNode == 0 {
// 		return fileSystem.directory, nil
// 	}
//
// 	directory, exists := fileSystem.directoryMap[iNode]
// 	if !exists {
// 		return nil, fmt.Errorf("directory with inode %d does not exist", iNode)
// 	}
//
// 	return directory, nil
// }

// func (fileSystem *FuseFileSystem) AddFile(parent *Directory, name string, videoUrl string, videoSize uint64) (*File, error) {
// 	if parent == nil {
// 		return nil, fmt.Errorf("parent directory is nil")
// 	}
//
// 	file := fileSystem.NewFile(name, videoUrl, videoSize)
//
// 	fileSystem.mu.Lock()
// 	parent.files[file.iNode] = file
// 	defer fileSystem.mu.Unlock()
//
// 	defer parent.Invalidate()
//
// 	return file, nil
// }

// func (fileSystem *FuseFileSystem) RemoveFile(parent *Directory, nodeId uint64) {
// 	fileSystem.mu.Lock()
// 	defer fileSystem.mu.Unlock()
//
// 	delete(parent.files, nodeId)
// }

// func (fileSystem *FuseFileSystem) RenameDirectory(directory *Directory, newName string) error {
// 	fileSystem.mu.Lock()
// 	defer fileSystem.mu.Unlock()
//
// 	return nil
// }
