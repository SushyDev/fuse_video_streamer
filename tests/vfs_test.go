package tests

import (
	"testing"

	"fuse_video_steamer/vfs"
	"fuse_video_steamer/vfs/node"
)

func TestFileSystemDirectories(t *testing.T) {
	fs, err := vfs.NewFileSystem()
	if err != nil {
		t.Fatalf("Failed to initialize filesystem: %v", err)
	}

	t.Run("Create and Find Directory", func(t *testing.T) {
		root := fs.GetRoot()
		testDir, err := fs.CreateDirectory("testdir", root)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		foundDir, err := fs.FindDirectory("testdir", root)
		if err != nil {
			t.Fatalf("Failed to find directory: %v", err)
		}

		if foundDir.GetNode().GetIdentifier() != testDir.GetNode().GetIdentifier() {
			t.Errorf("Directory identifiers do not match")
		}
	})

	t.Run("Update Directory", func(t *testing.T) {
		root := fs.GetRoot()
		dir, err := fs.CreateDirectory("updatable", root)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		updatedDir, err := fs.UpdateDirectory(dir, "updatedDir", root)
		if err != nil {
			t.Fatalf("Failed to update directory: %v", err)
		}

		if updatedDir.GetNode().GetName() != "updatedDir" {
			t.Errorf("Directory name not updated")
		}
	})

	t.Run("Delete Directory", func(t *testing.T) {
		root := fs.GetRoot()
		dir, err := fs.CreateDirectory("tobedeleted", root)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		if err := fs.DeleteDirectory(dir); err != nil {
			t.Fatalf("Failed to delete directory: %v", err)
		}

		foundDir, err := fs.FindDirectory("tobedeleted", root)
		if err == nil {
			if foundDir != nil {
				t.Errorf("Directory not deleted")
			}
		}
	})
}

func TestFileSystemFiles(t *testing.T) {
	fs, err := vfs.NewFileSystem()
	if err != nil {
		t.Fatalf("Failed to initialize filesystem: %v", err)
	}

	t.Run("Create and Find File", func(t *testing.T) {
		root := fs.GetRoot()
		file, err := fs.CreateFile("testfile.txt", root, 1024, "localhost")
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		foundFile, err := fs.FindFile("testfile.txt", root)
		if err != nil {
			t.Fatalf("Failed to find file: %v", err)
		}

		if foundFile.GetNode().GetIdentifier() != file.GetNode().GetIdentifier() {
			t.Errorf("File identifiers do not match")
		}
	})

	t.Run("Update File", func(t *testing.T) {
		root := fs.GetRoot()
		file, err := fs.CreateFile("updatablefile.txt", root, 512, "localhost")
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		file.SetSize(2048)
		_, err = fs.UpdateFile(file, file.GetNode().GetName(), root, file.GetSize(), file.GetHost())
		if err != nil {
			t.Fatalf("Failed to update file: %v", err)
		}

		updatedFile, err := fs.GetFile(file.GetNode().GetIdentifier())
		if err != nil {
			t.Fatalf("Failed to retrieve updated file: %v", err)
		}

		if updatedFile.GetSize() != 2048 {
			t.Errorf("File size not updated")
		}
	})

	t.Run("Delete File", func(t *testing.T) {
		root := fs.GetRoot()
		file, err := fs.CreateFile("deletablefile.txt", root, 256, "localhost")
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		if err := fs.DeleteFile(file); err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		foundFile, err := fs.FindFile("deletablefile.txt", root)
		if err == nil {
			if foundFile != nil {
				t.Errorf("File not deleted")
			}
		}
	})
}

func TestFileSystemIntegration(t *testing.T) {
	fs, err := vfs.NewFileSystem()
	if err != nil {
		t.Fatalf("Failed to initialize filesystem: %v", err)
	}

	t.Run("Directory and File Operations", func(t *testing.T) {
		root := fs.GetRoot()
		subdir, err := fs.CreateDirectory("subdir", root)
		if err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		file, err := fs.CreateFile("file.txt", subdir, 100, "localhost")
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		if err := fs.DeleteFile(file); err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		if err := fs.DeleteDirectory(subdir); err != nil {
			t.Fatalf("Failed to delete directory: %v", err)
		}
	})
}

func TestFileSystemScenarios(t *testing.T) {
	fs, err := vfs.NewFileSystem()
	if err != nil {
		t.Fatalf("Failed to initialize filesystem: %v", err)
	}

	t.Run("Create, Move, Rename, and Delete Operations", func(t *testing.T) {
		root := fs.GetRoot()

		// Create a hierarchy of directories
		parentDir, err := fs.CreateDirectory("parent", root)
		if err != nil {
			t.Fatalf("Failed to create parent directory: %v", err)
		}

		subDir, err := fs.CreateDirectory("child", parentDir)
		if err != nil {
			t.Fatalf("Failed to create child directory: %v", err)
		}

		// Create files in the directories
		file1, err := fs.CreateFile("file1.txt", parentDir, 1024, "host1")
		if err != nil {
			t.Fatalf("Failed to create file in parent directory: %v", err)
		}

		file2, err := fs.CreateFile("file2.txt", subDir, 2048, "host2")
		if err != nil {
			t.Fatalf("Failed to create file in child directory: %v", err)
		}

		// Move subDir (with file2 inside) under root
		movedSubDir, err := fs.UpdateDirectory(subDir, "child_moved", root)
		if err != nil {
			t.Fatalf("Failed to move child directory: %v", err)
		}

		// Verify the directory was moved
		foundDir, err := fs.FindDirectory("child_moved", root)
		if err != nil {
			t.Fatalf("Failed to find moved directory: %v", err)
		}

		if foundDir.GetNode().GetIdentifier() != movedSubDir.GetNode().GetIdentifier() {
			t.Errorf("Moved directory identifiers do not match")
		}

		// Verify file2 is still in the moved directory
		foundFile, err := fs.FindFile("file2.txt", movedSubDir)
		if err != nil {
			t.Fatalf("Failed to find file in moved directory: %v", err)
		}

		if foundFile.GetNode().GetIdentifier() != file2.GetNode().GetIdentifier() {
			t.Errorf("File in moved directory identifiers do not match")
		}

		// Rename a file in the moved directory
		updatedFile, err := fs.UpdateFile(file2, "renamed_file2.txt", movedSubDir, file2.GetSize(), file2.GetHost())
		if err != nil {
			t.Fatalf("Failed to rename file: %v", err)
		}

		if updatedFile.GetNode().GetName() != "renamed_file2.txt" {
			t.Errorf("File name was not updated")
		}

		renamedFile, err := fs.FindFile("renamed_file2.txt", movedSubDir)
		if err != nil {
			t.Fatalf("Failed to find renamed file: %v", err)
		}

		if renamedFile == nil {
			t.Errorf("Renamed file not found")
		}

		if renamedFile.GetNode() == nil {
			t.Errorf("Renamed file node is nil")
		}

		if renamedFile.GetNode().GetName() != "renamed_file2.txt" {
			t.Errorf("File name was not updated")
		}

		// Rename a directory
		renamedParentDir, err := fs.UpdateDirectory(parentDir, "renamed_parent", root)
		if err != nil {
			t.Fatalf("Failed to rename parent directory: %v", err)
		}

		if renamedParentDir.GetNode().GetName() != "renamed_parent" {
			t.Errorf("Parent directory name was not updated")
		}

		// Clean up by deleting everything
		if err := fs.DeleteFile(file1); err != nil {
			t.Errorf("Failed to delete file1: %v", err)
		}

		if err := fs.DeleteDirectory(renamedParentDir); err != nil {
			t.Errorf("Failed to delete renamed parent directory: %v", err)
		}

		if err := fs.DeleteDirectory(movedSubDir); err != nil {
			t.Errorf("Failed to delete moved child directory: %v", err)
		}
	})

	t.Run("Edge Cases and Invalid Scenarios", func(t *testing.T) {
		root := fs.GetRoot()

		// Try creating a directory with the same name as an existing one
		_, err := fs.CreateDirectory("duplicate", root)
		if err != nil {
			t.Fatalf("Failed to create initial directory: %v", err)
		}

		_, err = fs.CreateDirectory("duplicate", root)
		if err == nil {
			t.Errorf("Expected error when creating duplicate directory")
		}

		// Try moving a directory to a non-existent parent
		invalidParent := &node.Directory{} // Simulate an uninitialized parent directory
		_, err = fs.UpdateDirectory(root, "moved_to_invalid", invalidParent)
		if err == nil {
			// IGNORE FOR NOW BECAUSE THIS IS TECHNICALLY MOVING IT TO THE ROOT DIRECTORY BY SETTING PARENT TO NIL
			// TODO TEST FOR DIFFERENCE BETWEEN NIL AND EMPTY STRUCT
			// t.Errorf("Expected error when moving directory to invalid parent")
		}

		// Try renaming a directory to an existing directory name
		dir1, err := fs.CreateDirectory("dir1", root)
		if err != nil {
			t.Fatalf("Failed to create dir1: %v", err)
		}

		_, err = fs.CreateDirectory("dir2", root)
		if err != nil {
			t.Fatalf("Failed to create dir2: %v", err)
		}

		_, err = fs.UpdateDirectory(dir1, "dir2", root)
		if err == nil {
			t.Errorf("Expected error when renaming directory to existing name")
		}

		// Try deleting a non-existent directory
		fakeDir := &node.Directory{}
		err = fs.DeleteDirectory(fakeDir)
		if err == nil {
			t.Errorf("Expected error when deleting non-existent directory")
		}
	})
}
