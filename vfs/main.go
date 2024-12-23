package vfs

import (
	"fmt"
	"fuse_video_steamer/vfs/index"
	"fuse_video_steamer/vfs/node"
	"fuse_video_steamer/vfs/service"
	"log"
)

type FileSystem struct {
	root *node.Directory

	nodeService      *service.NodeService
	directoryService *service.DirectoryService
	fileService      *service.FileService

	// mu sync.RWMutex // TODO
}

func NewFileSystem() (*FileSystem, error) {
	index, err := index.New()
	if err != nil {
		return nil, fmt.Errorf("Failed to create index\n%w", err)
	}

	db := index.GetDB()

	nodeService := service.NewNodeService()

	directoryService := service.NewDirectoryService(db, nodeService)
	fileService := service.NewFileService(db, nodeService)

	fileSystem := &FileSystem{
		nodeService:      nodeService,
		directoryService: directoryService,
		fileService:      fileService,
	}

	root, err := fileSystem.FindOrCreateDirectory("root", nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to get root directory\n%w", err)
	}

	fileSystem.root = root

	return fileSystem, nil
}

func (fileSystem *FileSystem) GetRoot() *node.Directory {
	return fileSystem.root
}

// --- Directory

func (fileSystem *FileSystem) FindOrCreateDirectory(name string, parent *node.Directory) (*node.Directory, error) {
	directory, err := fileSystem.FindDirectory(name, parent)
	if err != nil {
		log.Printf("Failed to find directory %s\n", name)
		return nil, err
	}

	if directory != nil {
		return directory, nil
	}

	directory, err = fileSystem.CreateDirectory(name, parent)
	if err != nil {
		return nil, err
	}

	return directory, nil
}

func (fileSystem *FileSystem) FindDirectory(name string, parent *node.Directory) (*node.Directory, error) {
	directory, err := fileSystem.directoryService.FindDirectory(name, parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to find directory\n%w", err)
	}

	return directory, nil
}

func (fileSystem *FileSystem) CreateDirectory(name string, parent *node.Directory) (*node.Directory, error) {
	nodeId, err := fileSystem.directoryService.CreateDirectory(name, parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to create directory\n%w", err)
	}

	if nodeId == nil {
		return nil, fmt.Errorf("Failed to create directory\n")
	}

	directory, err := fileSystem.GetDirectory(*nodeId)
	if err != nil {
		return nil, fmt.Errorf("Failed to get directory\n%w", err)
	}

	return directory, nil
}

func (fileSystem *FileSystem) DeleteDirectory(directory *node.Directory) error {
	node := directory.GetNode()

	if node == nil {
		return fmt.Errorf("Node is nil")
	}

	err := fileSystem.directoryService.DeleteDirectory(node.GetIdentifier())
	if err != nil {
		return fmt.Errorf("Failed to delete directory\n%w", err)
	}

	return nil
}

func (fileSystem *FileSystem) UpdateDirectory(directory *node.Directory, name string, parent *node.Directory) (*node.Directory, error) {
	nodeId, err := fileSystem.directoryService.UpdateDirectory(directory.GetNode().GetIdentifier(), name, parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to update directory\n%w", err)
	}

	newDirectory, err := fileSystem.GetDirectory(*nodeId)
	if err != nil {
		return nil, fmt.Errorf("Failed to get directory\n%w", err)
	}

	return newDirectory, nil
}

func (fileSystem *FileSystem) GetDirectory(identifier uint64) (*node.Directory, error) {
	directory, err := fileSystem.directoryService.GetDirectory(identifier)
	if err != nil {
		return nil, fmt.Errorf("Failed to get directory\n%w", err)
	}

	return directory, nil
}

func (fileSystem *FileSystem) FindDirector(name string, parent *node.Directory) (*node.Directory, error) {
	directory, err := fileSystem.directoryService.FindDirectory(name, parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to find directory\n%w", err)
	}

	return directory, nil
}

func (fileSystem *FileSystem) GetChildNodes(parent *node.Directory) ([]*node.Node, error) {
	nodes, err := fileSystem.directoryService.GetChildNodes(parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to get directories\n%w", err)
	}

	return nodes, nil
}

// func (fileSystem *FileSystem) FindDirectory(name string) (*node.Directory, error) {
//     directory, err := fileSystem.index.FindDirectory(name)
//     if err != nil {
//         return nil, fmt.Errorf("Failed to find directory\n%w", err)
//     }
//
//     return directory, nil
// }

// --- File

func (fileSystem *FileSystem) CreateFile(name string, parent *node.Directory, size uint64, host string) (*node.File, error) {
	identifier, err := fileSystem.fileService.CreateFile(name, parent, size, host)
	if err != nil {
		return nil, fmt.Errorf("Failed to register file\n%w", err)
	}

	file, err := fileSystem.GetFile(*identifier)
	if err != nil {
		return nil, fmt.Errorf("Failed to get file\n%w", err)
	}

	return file, nil
}

func (fileSystem *FileSystem) DeleteFile(file *node.File) error {
	err := fileSystem.fileService.DeleteFile(file.GetNode().GetIdentifier())
	if err != nil {
		return fmt.Errorf("Failed to deregister file\n%w", err)
	}

	return nil
}

func (fileSystem *FileSystem) UpdateFile(file *node.File, name string, parent *node.Directory, size uint64, host string) (*node.File, error) {
	identifier, err := fileSystem.fileService.UpdateFile(
		file.GetNode().GetIdentifier(),
		name,
		parent,
		size,
		host,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to update file\n%w", err)
	}

	newFile, err := fileSystem.GetFile(*identifier)
	if err != nil {
		return nil, fmt.Errorf("Failed to get file\n%w", err)
	}

	return newFile, nil
}

func (fileSystem *FileSystem) GetFile(identifier uint64) (*node.File, error) {
	file, err := fileSystem.fileService.GetFile(identifier)
	if err != nil {
		return nil, fmt.Errorf("Failed to get file\n%w", err)
	}

	return file, nil
}

func (fileSystem *FileSystem) FindFile(name string, parent *node.Directory) (*node.File, error) {
	file, err := fileSystem.fileService.FindFile(name, parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to find file by name\n%w", err)
	}

	return file, nil
}

func (fileSystem *FileSystem) GetFiles(parent *node.Directory) ([]*node.File, error) {
	files, err := fileSystem.fileService.GetFiles(parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to get files\n%w", err)
	}

	return files, nil
}

// func (fileSystem *FileSystem) FindFile(name string) (*node.File, error) {
//     file, err := fileSystem.index.FindFile(name)
//     if err != nil {
//         return nil, fmt.Errorf("Failed to find file\n%w", err)
//     }
//
//     return file, nil
// }
