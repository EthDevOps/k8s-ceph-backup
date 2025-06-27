package main

import (
	"fmt"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate tool configuration and dependencies",
	Long:  `Check that all required dependencies and configurations are properly set up.`,
	Run: func(cmd *cobra.Command, args []string) {
		runValidate()
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate() {
	log.Info("Validating k8s-ceph-backup configuration and dependencies...")

	var errors []string

	// Check Kubernetes connectivity
	log.Info("Checking Kubernetes connectivity...")
	if _, err := createK8sClient(); err != nil {
		errors = append(errors, fmt.Sprintf("Kubernetes client: %v", err))
	} else {
		log.Info("✓ Kubernetes client connection successful")
	}

	// Check RBD command
	log.Info("Checking rbd command availability...")
	cephClient := NewCephClient()
	if cephClient.rbdPath == "" {
		cephClient.rbdPath = "rbd"
	}
	if _, err := exec.LookPath(cephClient.rbdPath); err != nil {
		errors = append(errors, fmt.Sprintf("rbd command not found: %v", err))
	} else {
		log.Info("✓ rbd command available")
	}

	// Check GPG setup
	log.Info("Checking GPG configuration...")
	gpgClient := NewGPGClient()
	if err := gpgClient.ValidateRecipient(); err != nil {
		errors = append(errors, fmt.Sprintf("GPG configuration: %v", err))
	} else {
		log.Info("✓ GPG configuration valid")
	}

	// Check MinIO connectivity
	log.Info("Checking MinIO connectivity...")
	minioClient := NewMinioClient()
	if _, err := minioClient.ListObjects(""); err != nil {
		errors = append(errors, fmt.Sprintf("MinIO connectivity: %v", err))
	} else {
		log.Info("✓ MinIO connectivity successful")
	}

	// Report results
	if len(errors) > 0 {
		log.Error("Validation failed with the following errors:")
		for _, err := range errors {
			log.Errorf("  - %s", err)
		}
		fmt.Printf("\nValidation completed with %d error(s). Please fix the issues above.\n", len(errors))
	} else {
		log.Info("✓ All validations passed successfully!")
		fmt.Println("\nValidation completed successfully. The tool is ready to use.")
	}
}