package main

import (
	"debrid_drive/vfs"
	// "debrid_drive/config"
	// "debrid_drive/stream"
	// "fmt"
	// "io"
)

// Implement the Read and Seek methods as per your previous implementation...

func main() {
	vfs.Start()

	// // Create a new SeekableVideoStream
	// videoStream, err := stream.NewPartialReader(config.VideoURL)
	// if err != nil {
	// 	fmt.Println("Error creating video stream:", err)
	// 	return
	// }
	// defer videoStream.Close()
	//
	// // Test seeking and reading
	// positions := []int64{0, 1024 * 1024, 5 * 1024 * 1024} // Test various positions
	// for _, pos := range positions {
	// 	// Seek to the position
	// 	_, err = videoStream.Seek(pos, io.SeekStart)
	// 	if err != nil {
	// 		fmt.Printf("Error seeking to position %d: %v\n", pos, err)
	// 		continue
	// 	}
	//
	// 	// Read data
	// 	buffer := make([]byte, 4096) // Read 4 KB
	// 	n, err := videoStream.Read(buffer)
	// 	if err != nil && err != io.EOF {
	// 		fmt.Printf("Error reading from position %d: %v\n", pos, err)
	// 		continue
	// 	}
	//
	// 	fmt.Printf("Read %d bytes from position %d\n", n, pos)
	// }
	//
	// // Test reading the last part of the video
	// _, err = videoStream.Seek(-4096, io.SeekEnd) // Seek to 4KB before the end
	// if err != nil {
	// 	fmt.Println("Error seeking to the end:", err)
	// 	return
	// }
	//
	// // Read the last 4 KB
	// buffer := make([]byte, 4096)
	// n, err := videoStream.Read(buffer)
	// if err != nil && err != io.EOF {
	// 	fmt.Printf("Error reading from the end: %v\n", err)
	// 	return
	// }
	//
	// fmt.Printf("Read %d bytes from the end of the video\n", n)
	//
	//    videoStream.CheckIsMKV()
	//
	// // Check if the stream is MKV
	// if !videoStream.IsMkv {
	// 	fmt.Println("The stream is not a valid MKV file.")
	// 	return
	// }
	// fmt.Println("The stream is a valid MKV file.")
}
