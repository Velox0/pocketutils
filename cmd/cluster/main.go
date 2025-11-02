package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const SEPARATOR = 0xFFFFFFFF // 32-bit separator

type FileMetadata struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type Separator struct {
	Value uint32
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: cluster <directory|cluster-file>")
		os.Exit(1)
	}

	path := os.Args[1]
	info, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if info.IsDir() {
		// Pack directory into cluster
		if err := packDirectory(path); err != nil {
			fmt.Printf("Error packing: %v\n", err)
			os.Exit(1)
		}
	} else if strings.HasSuffix(path, ".cluster") {
		// Unpack cluster file
		if err := unpackCluster(path); err != nil {
			fmt.Printf("Error unpacking: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Error: Must provide a directory or .cluster file")
		os.Exit(1)
	}
}

func packDirectory(dirPath string) error {
	// Collect all files
	var files []FileMetadata
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(dirPath, path)
			files = append(files, FileMetadata{
				Path: relPath,
				Size: info.Size(),
			})
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Create output file
	outputPath := filepath.Base(dirPath) + ".cluster"
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Serialize metadata
	metadataJSON, err := json.Marshal(files)
	if err != nil {
		return err
	}

	// Write metadata size (4 bytes)
	metadataSize := uint32(len(metadataJSON))
	if err := binary.Write(outFile, binary.LittleEndian, metadataSize); err != nil {
		return err
	}

	// Write metadata
	if _, err := outFile.Write(metadataJSON); err != nil {
		return err
	}

	// Write separator
	sep := Separator{Value: SEPARATOR}
	if err := binary.Write(outFile, binary.LittleEndian, sep); err != nil {
		return err
	}

	// Write file contents
	for _, file := range files {
		fullPath := filepath.Join(dirPath, file.Path)
		inFile, err := os.Open(fullPath)
		if err != nil {
			return err
		}

		if _, err := io.Copy(outFile, inFile); err != nil {
			inFile.Close()
			return err
		}
		inFile.Close()
		fmt.Printf("Added: %s (%d bytes)\n", file.Path, file.Size)
	}

	fmt.Printf("\nCluster created: %s\n", outputPath)
	fmt.Printf("Total files: %d\n", len(files))
	return nil
}

func unpackCluster(clusterPath string) error {
	inFile, err := os.Open(clusterPath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	// Read metadata size
	var metadataSize uint32
	if err := binary.Read(inFile, binary.LittleEndian, &metadataSize); err != nil {
		return err
	}

	// Read metadata
	metadataJSON := make([]byte, metadataSize)
	if _, err := io.ReadFull(inFile, metadataJSON); err != nil {
		return err
	}

	var files []FileMetadata
	if err := json.Unmarshal(metadataJSON, &files); err != nil {
		return err
	}

	// Read separator
	var sep Separator
	if err := binary.Read(inFile, binary.LittleEndian, &sep); err != nil {
		return err
	}
	if sep.Value != SEPARATOR {
		return fmt.Errorf("invalid separator: expected %x, got %x", SEPARATOR, sep.Value)
	}

	// Create output directory
	outputDir := strings.TrimSuffix(filepath.Base(clusterPath), ".cluster")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Extract files
	for _, file := range files {
		outPath := filepath.Join(outputDir, file.Path)

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		// Create file
		outFile, err := os.Create(outPath)
		if err != nil {
			return err
		}

		// Copy exact number of bytes
		if _, err := io.CopyN(outFile, inFile, file.Size); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
		fmt.Printf("Extracted: %s (%d bytes)\n", file.Path, file.Size)
	}

	fmt.Printf("\nCluster unpacked to: %s\n", outputDir)
	fmt.Printf("Total files: %d\n", len(files))
	return nil
}
