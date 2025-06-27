package main

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var restoreCmd = &cobra.Command{
	Use:   "restore [backup-file-name] [target-pool] [target-image]",
	Short: "Restore a backup to a CEPH RBD image",
	Long: `Restore a backup from MinIO storage to a CEPH RBD image.
This command will:
1. Download the backup from MinIO
2. Decrypt with GPG
3. Decompress with gzip
4. Import to RBD using the specified pool and image name`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		runRestore(args[0], args[1], args[2])
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}

func runRestore(backupFile, targetPool, targetImage string) {
	log.Infof("Starting restore process for backup: %s", backupFile)
	log.Infof("Target: %s/%s", targetPool, targetImage)

	restoreService := NewRestoreService()
	if err := restoreService.Run(backupFile, targetPool, targetImage); err != nil {
		log.Fatal("Restore failed:", err)
	}

	log.Info("Restore completed successfully")
}

type RestoreService struct {
	minioClient *MinioClient
	gpgClient   *GPGClient
	cephClient  *CephClient
}

func NewRestoreService() *RestoreService {
	return &RestoreService{
		minioClient: NewMinioClient(),
		gpgClient:   NewGPGClient(),
		cephClient:  NewCephClient(),
	}
}

func (rs *RestoreService) Run(backupFile, targetPool, targetImage string) error {
	tempDir := viper.GetString("backup.temp_dir")
	if tempDir == "" {
		tempDir = "/tmp/k8s-ceph-backup"
	}

	downloadPath := filepath.Join(tempDir, backupFile)
	
	log.Info("Downloading backup from MinIO...")
	if err := rs.minioClient.DownloadFile(backupFile, downloadPath); err != nil {
		return fmt.Errorf("failed to download backup: %w", err)
	}
	defer RemoveFile(downloadPath)

	log.Info("Decrypting backup...")
	decryptedPath, err := rs.gpgClient.DecryptFile(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to decrypt backup: %w", err)
	}
	defer RemoveFile(decryptedPath)

	log.Info("Decompressing backup...")
	decompressedPath, err := DecompressFile(decryptedPath)
	if err != nil {
		return fmt.Errorf("failed to decompress backup: %w", err)
	}
	defer RemoveFile(decompressedPath)

	log.Info("Importing to RBD...")
	if err := rs.cephClient.ImportImage(targetPool, targetImage, decompressedPath); err != nil {
		return fmt.Errorf("failed to import RBD image: %w", err)
	}

	return nil
}