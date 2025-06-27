# K8s CEPH Backup Tool

A comprehensive backup tool for Kubernetes Persistent Volume Claims (PVCs) backed by CEPH CSI. This tool automates the process of backing up CEPH RBD images by:

1. Listing PVCs in a specified namespace
2. Extracting CEPH pool and image information from attached Persistent Volumes
3. Exporting RBD images using the `rbd` command
4. Compressing the exports with gzip
5. Encrypting with GPG
6. Uploading to MinIO/S3 storage

## Prerequisites

- Go 1.21 or later
- Access to a Kubernetes cluster with CEPH CSI
- `rbd` command-line tool installed and configured
- `gpg` command-line tool installed with encryption keys
- MinIO server or S3-compatible storage

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd k8s-ceph-backup-tool

# Build the binary
go build -o k8s-ceph-backup

# Or install directly
go install
```

## Configuration

Copy the example configuration and customize it:

```bash
cp config.yaml.example ~/.k8s-ceph-backup.yaml
```

Edit the configuration file with your specific settings:

```yaml
# CEPH settings
ceph:
  rbd_path: "rbd"
  config_path: "/etc/ceph/ceph.conf"
  keyring_path: "/etc/ceph/keyring"

# GPG settings
gpg:
  recipient: "backup@example.com"

# MinIO settings
minio:
  endpoint: "minio.example.com:9000"
  access_key: "your-access-key"
  secret_key: "your-secret-key"
  bucket_name: "k8s-ceph-backups"
```

## Usage

### Basic Usage

Backup all CEPH-backed PVCs in the default namespace:
```bash
./k8s-ceph-backup
```

Backup PVCs in a specific namespace:
```bash
./k8s-ceph-backup --namespace production
```

### Command Line Options

- `--namespace, -n`: Kubernetes namespace to backup (default: "default")
- `--config`: Path to configuration file (default: ~/.k8s-ceph-backup.yaml)
- `--verbose, -v`: Enable verbose logging
- `--help, -h`: Show help

### Examples

```bash
# Backup production namespace with verbose output
./k8s-ceph-backup -n production -v

# Use custom config file
./k8s-ceph-backup --config /path/to/config.yaml

# Backup specific namespace
./k8s-ceph-backup --namespace database-cluster
```

## How It Works

1. **PVC Discovery**: The tool connects to Kubernetes and lists all PVCs in the specified namespace
2. **CEPH Detection**: For each bound PVC, it examines the associated PV to identify CEPH CSI volumes
3. **Metadata Extraction**: Extracts the CEPH pool name and RBD image name from the PV's CSI volume attributes
4. **RBD Export**: Uses the `rbd export` command to create a backup of the RBD image
5. **Compression**: Compresses the exported image using gzip to save space
6. **Encryption**: Encrypts the compressed file using GPG for security
7. **Upload**: Uploads the encrypted backup to MinIO/S3 storage

## Backup File Naming

Backup files are named using the following pattern:
```
{pvc-name}-{pool-name}-{image-name}.rbd.gz.gpg
```

Example: `app-data-rbd-pool-csi-vol-12345.rbd.gz.gpg`

## Security Considerations

- **GPG Encryption**: All backups are encrypted using GPG before upload
- **Access Control**: Ensure proper RBAC permissions for the Kubernetes service account
- **Credentials**: Store MinIO credentials securely (consider using Kubernetes secrets)
- **Network Security**: Use TLS for MinIO connections in production

## Monitoring and Logging

The tool provides comprehensive logging with different levels:
- `DEBUG`: Detailed operation information
- `INFO`: General operation status
- `WARN`: Non-critical issues
- `ERROR`: Critical errors

Enable verbose logging with the `-v` flag for troubleshooting.

## Kubernetes Permissions

The tool requires the following Kubernetes RBAC permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: k8s-ceph-backup
rules:
- apiGroups: [""]
  resources: ["persistentvolumeclaims"]
  verbs: ["get", "list"]
- apiGroups: [""]
  resources: ["persistentvolumes"]
  verbs: ["get"]
```

## Troubleshooting

### Common Issues

1. **"rbd command not found"**: Install ceph-common package
2. **"GPG recipient not found"**: Ensure GPG keys are properly imported
3. **"Access denied to MinIO"**: Verify MinIO credentials and bucket permissions
4. **"No CEPH volumes found"**: Check that PVCs are using CEPH CSI driver

### Debug Mode

Enable debug logging for detailed troubleshooting:
```bash
./k8s-ceph-backup -v
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.