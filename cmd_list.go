package main

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups in MinIO storage",
	Long:  `List all available backups stored in the configured MinIO bucket.`,
	Run: func(cmd *cobra.Command, args []string) {
		runList()
	},
}

var listPrefix string

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listPrefix, "prefix", "p", "", "Filter backups by prefix")
}

func runList() {
	log.Info("Listing available backups...")

	minioClient := NewMinioClient()
	objects, err := minioClient.ListObjects(listPrefix)
	if err != nil {
		log.Fatal("Failed to list backups:", err)
	}

	if len(objects) == 0 {
		fmt.Println("No backups found.")
		return
	}

	fmt.Printf("Found %d backup(s):\n\n", len(objects))
	fmt.Printf("%-50s %-20s %-20s %-20s\n", "Backup File", "PVC Name", "Pool", "Image")
	fmt.Println(strings.Repeat("-", 110))

	for _, object := range objects {
		pvc, pool, image := parseBackupFileName(object)
		fmt.Printf("%-50s %-20s %-20s %-20s\n", object, pvc, pool, image)
	}
}

func parseBackupFileName(filename string) (pvc, pool, image string) {
	base := strings.TrimSuffix(filename, ".rbd.gz.gpg")
	
	parts := strings.Split(base, "-")
	if len(parts) >= 3 {
		pvc = parts[0]
		pool = parts[1]
		image = strings.Join(parts[2:], "-")
	} else {
		pvc = filename
		pool = "unknown"
		image = "unknown"
	}
	
	return pvc, pool, image
}