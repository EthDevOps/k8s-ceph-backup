package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type CephClient struct {
	rbdPath    string
	configPath string
	keyringPath string
}

func NewCephClient() *CephClient {
	return &CephClient{
		rbdPath:     viper.GetString("ceph.rbd_path"),
		configPath:  viper.GetString("ceph.config_path"),
		keyringPath: viper.GetString("ceph.keyring_path"),
	}
}

func (c *CephClient) ExportImage(pool, imageName string) (string, error) {
	log.Infof("Exporting RBD image %s/%s", pool, imageName)

	if c.rbdPath == "" {
		c.rbdPath = "rbd"
	}

	exportDir := viper.GetString("backup.temp_dir")
	if exportDir == "" {
		exportDir = "/tmp/k8s-ceph-backup"
	}

	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	exportFile := filepath.Join(exportDir, fmt.Sprintf("%s-%s-%s.rbd", pool, imageName, timestamp))

	args := []string{"export"}
	
	if c.configPath != "" {
		args = append(args, "--conf", c.configPath)
	}
	
	if c.keyringPath != "" {
		args = append(args, "--keyring", c.keyringPath)
	}

	args = append(args, fmt.Sprintf("%s/%s", pool, imageName), exportFile)

	log.Debugf("Running rbd command: %s %v", c.rbdPath, args)

	cmd := exec.Command(c.rbdPath, args...)
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("rbd export failed: %w", err)
	}

	info, err := os.Stat(exportFile)
	if err != nil {
		return "", fmt.Errorf("failed to stat exported file: %w", err)
	}

	log.Infof("Successfully exported RBD image to %s (size: %d bytes)", exportFile, info.Size())
	return exportFile, nil
}

func (c *CephClient) ListImages(pool string) ([]string, error) {
	log.Debugf("Listing images in pool %s", pool)

	args := []string{"ls", pool}
	
	if c.configPath != "" {
		args = append(args, "--conf", c.configPath)
	}
	
	if c.keyringPath != "" {
		args = append(args, "--keyring", c.keyringPath)
	}

	cmd := exec.Command(c.rbdPath, args...)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list images in pool %s: %w", pool, err)
	}

	log.Debugf("Images in pool %s: %s", pool, string(output))
	return []string{string(output)}, nil
}

func (c *CephClient) ImageExists(pool, imageName string) (bool, error) {
	log.Debugf("Checking if image %s/%s exists", pool, imageName)

	args := []string{"info"}
	
	if c.configPath != "" {
		args = append(args, "--conf", c.configPath)
	}
	
	if c.keyringPath != "" {
		args = append(args, "--keyring", c.keyringPath)
	}

	args = append(args, fmt.Sprintf("%s/%s", pool, imageName))

	cmd := exec.Command(c.rbdPath, args...)
	
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 2 {
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to check image existence: %w", err)
	}

	return true, nil
}

func (c *CephClient) ImportImage(pool, imageName, importPath string) error {
	log.Infof("Importing RBD image %s/%s from %s", pool, imageName, importPath)

	if c.rbdPath == "" {
		c.rbdPath = "rbd"
	}

	args := []string{"import"}
	
	if c.configPath != "" {
		args = append(args, "--conf", c.configPath)
	}
	
	if c.keyringPath != "" {
		args = append(args, "--keyring", c.keyringPath)
	}

	args = append(args, importPath, fmt.Sprintf("%s/%s", pool, imageName))

	log.Debugf("Running rbd import command: %s %v", c.rbdPath, args)

	cmd := exec.Command(c.rbdPath, args...)
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rbd import failed: %w", err)
	}

	log.Infof("Successfully imported RBD image %s/%s", pool, imageName)
	return nil
}

func (c *CephClient) Cleanup(exportPath string) {
	log.Debugf("Cleaning up export file: %s", exportPath)
	if err := os.Remove(exportPath); err != nil {
		log.Warnf("Failed to remove export file %s: %v", exportPath, err)
	}
}