package api

import (
	"context"
	"net"

	"fuse_video_steamer/fuse"
	"fuse_video_steamer/logger"

	"google.golang.org/grpc"
)

type GrpcServer struct {
	fuse *fuse.Fuse

	UnimplementedFileSystemServer
}

var apiLogger, _ = logger.GetLogger(logger.ApiLogPath)

func (grpc *GrpcServer) AddDirectory(ctx context.Context, request *AddDirectoryRequest) (*DirectoryResponse, error) {
	fileSystem := grpc.fuse.GetVirtualFileSystem()

	parent := fileSystem.GetDirectory(request.ParentNodeId)
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

	newDirectory := fileSystem.NewDirectory(parent, request.Name)

	// server.InvalidateEntry(parent, newDirectory.GetName())

	return &DirectoryResponse{
		NodeId:  newDirectory.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func (grpc *GrpcServer) RenameDirectory(ctx context.Context, request *RenameDirectoryRequest) (*DirectoryResponse, error) {
	virtualFileSystem := grpc.fuse.GetVirtualFileSystem()

	directory := virtualFileSystem.GetDirectory(request.NodeId)
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

	virtualFileSystem.RenameDirectory(directory, request.Name, parent)

	// server.fuse.InvalidateEntry(directory.GetParent().GetIdentifier(), directory.GetName())

	return &DirectoryResponse{
		NodeId:  directory.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func (grpc *GrpcServer) RemoveDirectory(ctx context.Context, request *RemoveDirectoryRequest) (*DirectoryResponse, error) {
	virtualFileSystem := grpc.fuse.GetVirtualFileSystem()

	directory := virtualFileSystem.GetDirectory(request.NodeId)
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

	virtualFileSystem.RemoveDirectory(directory)

	// server.fuse.InvalidateEntry(directory.GetParent().GetIdentifier(), directory.GetName())

	return &DirectoryResponse{
		NodeId:  directory.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func (grpc *GrpcServer) AddFile(ctx context.Context, request *AddFileRequest) (*FileResponse, error) {
	virtualFileSystem := grpc.fuse.GetVirtualFileSystem()

	parent := virtualFileSystem.GetDirectory(request.ParentNodeId)
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

	newFile := virtualFileSystem.NewFile(parent, request.Name, request.VideoUrl, request.FetchUrl, request.FileSize)

	// server.fuse.InvalidateEntry(parent.GetIdentifier(), newFile.GetName())

	return &FileResponse{
		NodeId:  newFile.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func (grpc *GrpcServer) RenameFile(ctx context.Context, request *RenameFileRequest) (*FileResponse, error) {
	virtualFileSystem := grpc.fuse.GetVirtualFileSystem()

	file := virtualFileSystem.GetFile(request.NodeId)
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

	// server.fuse.InvalidateNode(file.GetIdentifier())

	return &FileResponse{
		NodeId:  file.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func (grpc *GrpcServer) RemoveFile(ctx context.Context, request *RemoveFileRequest) (*FileResponse, error) {
	virtualFileSystem := grpc.fuse.GetVirtualFileSystem()

	file := virtualFileSystem.GetFile(request.NodeId)
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

	virtualFileSystem.RemoveFile(file)

	// server.fuse.InvalidateNode(file.GetIdentifier())

	return &FileResponse{
		NodeId:  file.GetIdentifier(),
		Success: true,
		Error:   nil,
	}, nil
}

func Listen(fileSystem *fuse.Fuse) {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		apiLogger.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	RegisterFileSystemServer(grpcServer, &GrpcServer{
		fuse: fileSystem,
	})

	apiLogger.Infof("Starting server on port :50051")

	if err := grpcServer.Serve(listener); err != nil {
		apiLogger.Fatalf("failed to serve: %v", err)
	}
}
