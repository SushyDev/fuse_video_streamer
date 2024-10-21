package api

import (
	"context"
	"net"

	"debrid_drive/fuse"
	"debrid_drive/logger"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type GrpcServer struct {
	fileSystem *fuse.FuseFileSystem

	UnimplementedFileSystemServer
}

var apiLogger, _ = logger.GetLogger(logger.ApiLogPath)

func (server *GrpcServer) AddDirectory(ctx context.Context, request *AddDirectoryRequest) (*DirectoryResponse, error) {
	parentDirectory, err := server.fileSystem.VFS.GetDirectory(request.ParentNodeId)
	if err != nil {
        apiLogger.Error("Failed to get directory", zap.Error(err))

		return &DirectoryResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	newDirectory, err := parentDirectory.AddDirectory(request.Name)
	if err != nil {
        apiLogger.Error("Failed to add directory", zap.Error(err))

		return &DirectoryResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	server.fileSystem.InvalidateEntry(parentDirectory.ID, newDirectory.Name)

	return &DirectoryResponse{
		NodeId:  newDirectory.ID,
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) RenameDirectory(ctx context.Context, request *RenameDirectoryRequest) (*DirectoryResponse, error) {
	directory, err := server.fileSystem.VFS.GetDirectory(request.NodeId)
	if err != nil {
        apiLogger.Error("Failed to get directory", zap.Error(err))

		return &DirectoryResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	directory.Rename(request.Name)

	server.fileSystem.InvalidateEntry(directory.Parent.ID, directory.Name)

	return &DirectoryResponse{
		NodeId:  directory.ID,
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) RemoveDirectory(ctx context.Context, request *RemoveDirectoryRequest) (*DirectoryResponse, error) {
	directory, err := server.fileSystem.VFS.GetDirectory(request.NodeId)
	if err != nil {
        apiLogger.Error("Failed to get directory", zap.Error(err))

		return &DirectoryResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	err = directory.Parent.RemoveDirectory(directory.Name)
	if err != nil {
        apiLogger.Error("Failed to remove directory", zap.Error(err))

		return &DirectoryResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	server.fileSystem.InvalidateEntry(directory.Parent.ID, directory.Name)

	return &DirectoryResponse{
		NodeId:  directory.ID,
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) AddFile(ctx context.Context, request *AddFileRequest) (*FileResponse, error) {
	parentDirectory, err := server.fileSystem.VFS.GetDirectory(request.ParentNodeId)
	if err != nil {
        apiLogger.Error("Failed to get directory", zap.Error(err))

		return &FileResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	newFile, err := parentDirectory.AddFile(request.Name, request.VideoUrl, request.FetchUrl, request.FileSize)
	if err != nil {
        apiLogger.Error("Failed to add file", zap.Error(err))

		return &FileResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	server.fileSystem.InvalidateEntry(parentDirectory.ID, newFile.Name)

	return &FileResponse{
		NodeId:  newFile.ID,
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) RenameFile(ctx context.Context, request *RenameFileRequest) (*FileResponse, error) {
	file, err := server.fileSystem.VFS.GetFile(request.NodeId)
	if err != nil {
        apiLogger.Error("Failed to get file", zap.Error(err))

		return &FileResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	file.Rename(request.Name)

	server.fileSystem.InvalidateNode(file.ID)

	return &FileResponse{
		NodeId:  file.ID,
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) RemoveFile(ctx context.Context, request *RemoveFileRequest) (*FileResponse, error) {
	file, err := server.fileSystem.VFS.GetFile(request.NodeId)
	if err != nil {
        apiLogger.Error("Failed to get file", zap.Error(err))

		return &FileResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	err = file.Parent.RemoveFile(file.Name)
	if err != nil {
        apiLogger.Error("Failed to remove file", zap.Error(err))

		return &FileResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	server.fileSystem.InvalidateNode(file.ID)

	return &FileResponse{
		NodeId:  file.ID,
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
