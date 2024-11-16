package api

import (
	"context"
	"net"

	"debrid_drive/fuse"
	"debrid_drive/logger"

	"google.golang.org/grpc"
)

type GrpcServer struct {
	fileSystem *fuse.FuseFileSystem

	UnimplementedFileSystemServer
}

var apiLogger, _ = logger.GetLogger(logger.ApiLogPath)

func (server *GrpcServer) AddDirectory(ctx context.Context, request *AddDirectoryRequest) (*DirectoryResponse, error) {
	parent := server.fileSystem.VirtualFileSystem.GetDirectory(request.ParentNodeId)
	if parent == nil {
		message := "Could not find parent directory"

		apiLogger.Error(message)

		return &DirectoryResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: message,
			},
		}, nil
	}

	newDirectory := server.fileSystem.VirtualFileSystem.NewDirectory(parent, request.Name)

	server.fileSystem.InvalidateEntry(parent.GetIdentifier(), newDirectory.GetName())

	return &DirectoryResponse{
		NodeId:  newDirectory.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) RenameDirectory(ctx context.Context, request *RenameDirectoryRequest) (*DirectoryResponse, error) {
	directory := server.fileSystem.VirtualFileSystem.GetDirectory(request.NodeId)
	if directory == nil {
		message := "Could not find directory"

		apiLogger.Error(message)

		return &DirectoryResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: message,
			},
		}, nil
	}

	parent := directory.GetParent()

	server.fileSystem.VirtualFileSystem.RenameDirectory(directory, request.Name, parent)

	server.fileSystem.InvalidateEntry(directory.GetParent().GetIdentifier(), directory.GetName())

	return &DirectoryResponse{
		NodeId:  directory.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) RemoveDirectory(ctx context.Context, request *RemoveDirectoryRequest) (*DirectoryResponse, error) {
	directory := server.fileSystem.VirtualFileSystem.GetDirectory(request.NodeId)
	if directory == nil {
		message := "Could not find directory"

		apiLogger.Error(message)

		return &DirectoryResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: message,
			},
		}, nil
	}

	server.fileSystem.VirtualFileSystem.RemoveDirectory(directory)

	server.fileSystem.InvalidateEntry(directory.GetParent().GetIdentifier(), directory.GetName())

	return &DirectoryResponse{
		NodeId:  directory.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) AddFile(ctx context.Context, request *AddFileRequest) (*FileResponse, error) {
	parent := server.fileSystem.VirtualFileSystem.GetDirectory(request.ParentNodeId)
	if parent == nil {
		message := "Could not find parent directory"

		apiLogger.Error(message)

		return &FileResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: message,
			},
		}, nil
	}

	newFile := server.fileSystem.VirtualFileSystem.NewFile(parent, request.Name, request.VideoUrl, request.FetchUrl, request.FileSize)

	server.fileSystem.InvalidateEntry(parent.GetIdentifier(), newFile.GetName())

	return &FileResponse{
		NodeId:  newFile.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) RenameFile(ctx context.Context, request *RenameFileRequest) (*FileResponse, error) {
	file := server.fileSystem.VirtualFileSystem.GetFile(request.NodeId)
	if file == nil {
		message := "Could not find file"

		apiLogger.Error(message)

		return &FileResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: message,
			},
		}, nil
	}

	file.Rename(request.Name)

	server.fileSystem.InvalidateNode(file.GetIdentifier())

	return &FileResponse{
		NodeId:  file.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) RemoveFile(ctx context.Context, request *RemoveFileRequest) (*FileResponse, error) {
	file := server.fileSystem.VirtualFileSystem.GetFile(request.NodeId)
	if file == nil {
		message := "Could not find file"

		apiLogger.Error(message)

		return &FileResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: message,
			},
		}, nil
	}

	server.fileSystem.VirtualFileSystem.RemoveFile(file)

	server.fileSystem.InvalidateNode(file.GetIdentifier())

	return &FileResponse{
		NodeId:  file.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func Listen(fileSystem *fuse.FuseFileSystem) {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		apiLogger.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	RegisterFileSystemServer(grpcServer, &GrpcServer{
		fileSystem: fileSystem,
	})

	apiLogger.Infof("Starting server on port :50051")

	if err := grpcServer.Serve(listener); err != nil {
		apiLogger.Fatalf("failed to serve: %v", err)
	}
}
