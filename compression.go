package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func CompressFile(inputPath string) (string, error) {
	log.Debugf("Compressing file: %s", inputPath)

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	outputPath := inputPath + ".gz"
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	gzipWriter := gzip.NewWriter(outputFile)
	defer gzipWriter.Close()

	gzipWriter.Header.Name = filepath.Base(inputPath)

	bytesWritten, err := io.Copy(gzipWriter, inputFile)
	if err != nil {
		return "", fmt.Errorf("failed to compress file: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize compression: %w", err)
	}

	if err := outputFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close output file: %w", err)
	}

	inputInfo, err := inputFile.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat input file: %w", err)
	}

	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat output file: %w", err)
	}

	compressionRatio := float64(outputInfo.Size()) / float64(inputInfo.Size()) * 100

	log.Infof("Compressed %s: %d bytes -> %d bytes (%.1f%% of original)", 
		inputPath, inputInfo.Size(), bytesWritten, compressionRatio)

	return outputPath, nil
}

func DecompressFile(inputPath string) (string, error) {
	log.Debugf("Decompressing file: %s", inputPath)

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	gzipReader, err := gzip.NewReader(inputFile)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	outputPath := strings.TrimSuffix(inputPath, ".gz")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	bytesWritten, err := io.Copy(outputFile, gzipReader)
	if err != nil {
		return "", fmt.Errorf("failed to decompress file: %w", err)
	}

	log.Infof("Decompressed %s: %d bytes written", inputPath, bytesWritten)

	return outputPath, nil
}

func RemoveFile(path string) error {
	log.Debugf("Removing file: %s", path)
	return os.Remove(path)
}