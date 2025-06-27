#!/bin/sh
set -e

# Import GPG key if it exists
if [ -f /root/.gnupg/pubkey.asc ]; then
    echo "Importing GPG key from /root/.gnupg/pubkey.asc"
    gpg --import /root/.gnupg/pubkey.asc
else
    echo "Warning: GPG key not found at /root/.gnupg/pubkey.asc"
fi

# Execute k8s-ceph-backup with all provided arguments
exec ./k8s-ceph-backup "$@"