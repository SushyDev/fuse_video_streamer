package vfs

import (
	"fmt"
	"fuse_video_steamer/vfs/database"
	"fuse_video_steamer/vfs/node"
	"fuse_video_steamer/vfs/service"
	"log"
	"sync"
)

type FileSystem struct {
	root *node.Directory

	nodeService      *service.NodeService
	directoryService *service.DirectoryService
	fileService      *service.FileService

	mu sync.RWMutex
}

func NewFileSystem() (*FileSystem, error) {
	database, err := database.New()
	if err != nil {
		return nil, fmt.Errorf("Failed to create index\n%w", err)
	}

	nodeService := service.NewNodeService()
	directoryService := service.NewDirectoryService(database, nodeService)
	fileService := service.NewFileService(database, nodeService)

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
	fileSystem.mu.Lock()
	defer fileSystem.mu.Unlock()

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
	fileSystem.mu.Lock()
	defer fileSystem.mu.Unlock()

	return fileSystem.directoryService.FindDirectory(name, parent)
}

func (fileSystem *FileSystem) CreateDirectory(name string, parent *node.Directory) (*node.Directory, error) {
	fileSystem.mu.Lock()

	nodeId, err := fileSystem.directoryService.CreateDirectory(name, parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to create directory\n%w", err)
	}

	if nodeId == nil {
		return nil, fmt.Errorf("Failed to create directory\n")
	}

	fileSystem.mu.Unlock()

	return fileSystem.GetDirectory(*nodeId)
}

func (fileSystem *FileSystem) DeleteDirectory(directory *node.Directory) error {
	fileSystem.mu.Lock()
	defer fileSystem.mu.Unlock()

	node := directory.GetNode()
	if node == nil {
		return fmt.Errorf("Node is nil")
	}

	return fileSystem.directoryService.DeleteDirectory(node.GetIdentifier())
}

func (fileSystem *FileSystem) UpdateDirectory(directory *node.Directory, name string, parent *node.Directory) (*node.Directory, error) {
	fileSystem.mu.Lock()

	node := directory.GetNode()
	if node == nil {
		return nil, fmt.Errorf("Node is nil")
	}

	err := fileSystem.directoryService.UpdateDirectory(node.GetIdentifier(), name, parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to update directory\n%w", err)
	}

	fileSystem.mu.Unlock()

	return fileSystem.GetDirectory(node.GetIdentifier())
}

func (fileSystem *FileSystem) GetDirectory(identifier uint64) (*node.Directory, error) {
	fileSystem.mu.RLock()
	defer fileSystem.mu.RUnlock()

	return fileSystem.directoryService.GetDirectory(identifier)
}

func (fileSystem *FileSystem) GetChildNode(name string, parent *node.Directory) (*node.Node, error) {
	fileSystem.mu.RLock()
	defer fileSystem.mu.RUnlock()

	return fileSystem.directoryService.GetChildNode(name, parent)
}

func (fileSystem *FileSystem) GetChildNodes(parent *node.Directory) ([]*node.Node, error) {
	fileSystem.mu.RLock()
	defer fileSystem.mu.RUnlock()

	return fileSystem.directoryService.GetChildNodes(parent)
}

// --- File

func (fileSystem *FileSystem) CreateFile(name string, parent *node.Directory, size uint64, host string) (*node.File, error) {
	fileSystem.mu.Lock()

	identifier, err := fileSystem.fileService.CreateFile(name, parent, size, host)
	if err != nil {
		return nil, fmt.Errorf("Failed to register file\n%w", err)
	}

	fileSystem.mu.Unlock()

	return fileSystem.GetFile(*identifier)
}

func (fileSystem *FileSystem) DeleteFile(file *node.File) error {
	fileSystem.mu.Lock()
	defer fileSystem.mu.Unlock()

	return fileSystem.fileService.DeleteFile(file.GetNode().GetIdentifier())
}

func (fileSystem *FileSystem) UpdateFile(file *node.File, name string, parent *node.Directory, size uint64, host string) (*node.File, error) {
	fileSystem.mu.Lock()

	node := file.GetNode()
	if node == nil {
		return nil, fmt.Errorf("Node is nil")
	}

	err := fileSystem.fileService.UpdateFile(
		node.GetIdentifier(),
		name,
		parent,
		size,
		host,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to update file\n%w", err)
	}

	fileSystem.mu.Unlock()

	return fileSystem.GetFile(node.GetIdentifier())
}

func (fileSystem *FileSystem) GetFile(identifier uint64) (*node.File, error) {
	fileSystem.mu.RLock()
	defer fileSystem.mu.RUnlock()

	return fileSystem.fileService.GetFile(identifier)
}

func (fileSystem *FileSystem) FindFile(name string, parent *node.Directory) (*node.File, error) {
	fileSystem.mu.RLock()
	defer fileSystem.mu.RUnlock()

	return fileSystem.fileService.FindFile(name, parent)
}

func (fileSystem *FileSystem) GetFiles(parent *node.Directory) ([]*node.File, error) {
	fileSystem.mu.RLock()
	defer fileSystem.mu.RUnlock()

	return fileSystem.fileService.GetFiles(parent)
}
