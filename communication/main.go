package communication

import (
	"context"
	"log"
	"net"

	"debrid_drive/vfs"
	"google.golang.org/grpc"
)

type GrpcServer struct {
	fileSystem *vfs.FileSystem

	UnimplementedFileSystemServer
}

func (server *GrpcServer) AddDirectory(ctx context.Context, request *AddDirectoryRequest) (*DirectoryResponse, error) {
	parentDirectory, err := server.fileSystem.GetDirectory(request.ParentNodeId)
	if err != nil {
		return &DirectoryResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	newDirectory, err := parentDirectory.AddSubDirectory(request.Name)
	if err != nil {
		return &DirectoryResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	return &DirectoryResponse{
		NodeId:  newDirectory.ID,
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) RenameDirectory(ctx context.Context, request *RenameDirectoryRequest) (*DirectoryResponse, error) {
	directory, err := server.fileSystem.GetDirectory(request.NodeId)
	if err != nil {
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

	return &DirectoryResponse{
		NodeId:  directory.ID,
		Success: true,
		Error:   nil,
	}, nil
}

func (server *GrpcServer) AddFile(ctx context.Context, request *AddFileRequest) (*FileResponse, error) {
	parentDirectory, err := server.fileSystem.GetDirectory(request.ParentNodeId)
	if err != nil {
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
		return &FileResponse{
			NodeId:  0,
			Success: false,
			Error: &Error{
				Code:    1,
				Message: err.Error(),
			},
		}, err
	}

	return &FileResponse{
		NodeId:  newFile.ID,
		Success: true,
		Error:   nil,
	}, nil
}

func Listen(fileSystem *vfs.FileSystem) {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	RegisterFileSystemServer(grpcServer, &GrpcServer{
		fileSystem: fileSystem,
	})

	log.Println("Starting server on port :50051")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
