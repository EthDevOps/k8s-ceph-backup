package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type CephImage struct {
	Pool      string
	ImageName string
	PVCName   string
	PVName    string
}

type BackupService struct {
	k8sClient   kubernetes.Interface
	cephClient  *CephClient
	minioClient *MinioClient
	gpgClient   *GPGClient
}

func NewBackupService() *BackupService {
	k8sClient, err := createK8sClient()
	if err != nil {
		log.Fatal("Failed to create Kubernetes client:", err)
	}

	return &BackupService{
		k8sClient:   k8sClient,
		cephClient:  NewCephClient(),
		minioClient: NewMinioClient(),
		gpgClient:   NewGPGClient(),
	}
}

func createK8sClient() (kubernetes.Interface, error) {
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if err != nil {
		log.Debug("Not running in cluster, trying kubeconfig")
		
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create k8s config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return clientset, nil
}

func (bs *BackupService) Run(namespace string) error {
	log.Infof("Starting backup for namespace: %s", namespace)

	pvcs, err := bs.listPVCs(namespace)
	if err != nil {
		return fmt.Errorf("failed to list PVCs: %w", err)
	}

	log.Infof("Found %d PVCs in namespace %s", len(pvcs.Items), namespace)

	var cephImages []CephImage
	for _, pvc := range pvcs.Items {
		if pvc.Status.Phase != corev1.ClaimBound {
			log.Warnf("Skipping PVC %s: not bound", pvc.Name)
			continue
		}

		if pvc.Spec.VolumeName == "" {
			log.Warnf("Skipping PVC %s: no volume name", pvc.Name)
			continue
		}

		cephImage, err := bs.extractCephInfo(pvc)
		if err != nil {
			log.Errorf("Failed to extract CEPH info for PVC %s: %v", pvc.Name, err)
			continue
		}

		if cephImage != nil {
			cephImages = append(cephImages, *cephImage)
		}
	}

	log.Infof("Found %d CEPH-backed PVCs to backup", len(cephImages))

	for _, image := range cephImages {
		if err := bs.backupImage(image); err != nil {
			log.Errorf("Failed to backup image %s/%s: %v", image.Pool, image.ImageName, err)
			continue
		}
	}

	return nil
}

func (bs *BackupService) listPVCs(namespace string) (*corev1.PersistentVolumeClaimList, error) {
	return bs.k8sClient.CoreV1().PersistentVolumeClaims(namespace).List(
		context.TODO(), 
		metav1.ListOptions{},
	)
}

func (bs *BackupService) extractCephInfo(pvc corev1.PersistentVolumeClaim) (*CephImage, error) {
	pv, err := bs.k8sClient.CoreV1().PersistentVolumes().Get(
		context.TODO(), 
		pvc.Spec.VolumeName, 
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get PV %s: %w", pvc.Spec.VolumeName, err)
	}

	if pv.Spec.CSI == nil {
		log.Debugf("PVC %s: not a CSI volume", pvc.Name)
		return nil, nil
	}

	if !strings.Contains(pv.Spec.CSI.Driver, "ceph") {
		log.Debugf("PVC %s: not a CEPH CSI volume (driver: %s)", pvc.Name, pv.Spec.CSI.Driver)
		return nil, nil
	}

	volumeAttributes := pv.Spec.CSI.VolumeAttributes
	if volumeAttributes == nil {
		return nil, fmt.Errorf("no volume attributes found in PV %s", pv.Name)
	}

	pool, ok := volumeAttributes["pool"]
	if !ok {
		return nil, fmt.Errorf("pool not found in volume attributes for PV %s", pv.Name)
	}

	imageName, ok := volumeAttributes["imageName"]
	if !ok {
		return nil, fmt.Errorf("imageName not found in volume attributes for PV %s", pv.Name)
	}

	log.Infof("Found CEPH image: pool=%s, image=%s for PVC %s", pool, imageName, pvc.Name)

	return &CephImage{
		Pool:      pool,
		ImageName: imageName,
		PVCName:   pvc.Name,
		PVName:    pv.Name,
	}, nil
}

func (bs *BackupService) backupImage(image CephImage) error {
	log.Infof("Starting backup for image %s/%s (PVC: %s)", image.Pool, image.ImageName, image.PVCName)

	exportPath, err := bs.cephClient.ExportImage(image.Pool, image.ImageName)
	if err != nil {
		return fmt.Errorf("failed to export RBD image: %w", err)
	}
	defer bs.cephClient.Cleanup(exportPath)

	compressedPath, err := bs.compressFile(exportPath)
	if err != nil {
		return fmt.Errorf("failed to compress file: %w", err)
	}
	defer bs.cleanup(compressedPath)

	encryptedPath, err := bs.gpgClient.EncryptFile(compressedPath)
	if err != nil {
		return fmt.Errorf("failed to encrypt file: %w", err)
	}
	defer bs.cleanup(encryptedPath)

	objectName := fmt.Sprintf("%s-%s-%s.rbd.gz.gpg", image.PVCName, image.Pool, image.ImageName)
	if err := bs.minioClient.UploadFile(encryptedPath, objectName); err != nil {
		return fmt.Errorf("failed to upload to MinIO: %w", err)
	}

	log.Infof("Successfully backed up image %s/%s to %s", image.Pool, image.ImageName, objectName)
	return nil
}

func (bs *BackupService) compressFile(inputPath string) (string, error) {
	log.Debug("Compressing file:", inputPath)
	return CompressFile(inputPath)
}

func (bs *BackupService) cleanup(path string) {
	if err := RemoveFile(path); err != nil {
		log.Warnf("Failed to cleanup file %s: %v", path, err)
	}
}