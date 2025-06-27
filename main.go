package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	log "github.com/sirupsen/logrus"
)

var (
	cfgFile   string
	namespace string
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "k8s-ceph-backup",
	Short: "Backup CEPH CSI backed PVCs in Kubernetes",
	Long: `A tool to backup Kubernetes Persistent Volume Claims that are backed by CEPH CSI.
This tool will:
1. List PVCs in a namespace
2. Extract CEPH pool and image information from attached PVs
3. Export RBD images using rbd command
4. Compress with gzip
5. Encrypt with GPG
6. Upload to MinIO storage`,
	Run: func(cmd *cobra.Command, args []string) {
		runBackup()
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.k8s-ceph-backup.yaml)")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace to backup PVCs from")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	viper.BindPFlag("namespace", rootCmd.PersistentFlags().Lookup("namespace"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".k8s-ceph-backup")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Info("Using config file:", viper.ConfigFileUsed())
	}

	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}
}

func runBackup() {
	log.Info("Starting CEPH CSI PVC backup process")
	
	backupService := NewBackupService()
	if err := backupService.Run(viper.GetString("namespace")); err != nil {
		log.Fatal("Backup failed:", err)
	}
	
	log.Info("Backup completed successfully")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}