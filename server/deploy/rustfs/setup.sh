#!/bin/bash
set -e

# Configuration
S3_ENDPOINT="http://rustfs:9000"
ALIAS="myminio"

echo "Starting S3 Setup..."

# Wait for Service
echo "Waiting for S3 API at $S3_ENDPOINT..."
until curl -s --head --request GET "$S3_ENDPOINT/health" > /dev/null 2>&1 || curl -s "$S3_ENDPOINT" > /dev/null 2>&1; do
# RustFS might not support minio health check endpoint, just check connectivity
  echo "Waiting for S3 service..."
  sleep 2
done
echo "S3 API is reachable."

# Configure mc alias
# SAMSA_RUSTFS_ACCESS_KEY and SAMSA_RUSTFS_SECRET_KEY should be passed as env vars
echo "Configuring mc alias..."
mc alias set "$ALIAS" "$S3_ENDPOINT" "$SAMSA_RUSTFS_ACCESS_KEY" "$SAMSA_RUSTFS_SECRET_KEY"

# Create Buckets
if [ -n "$SAMSA_AWS_S3_BUCKETS" ]; then
    # BUCKETS are comma separated
    IFS=',' read -ra BUCKET_LIST <<< "$SAMSA_AWS_S3_BUCKETS"

    for BUCKET in "${BUCKET_LIST[@]}"; do
        # Trim whitespace
        BUCKET=$(echo "$BUCKET" | xargs)

        echo "Ensuring bucket exists: $BUCKET"
        if ! mc ls "$ALIAS/$BUCKET" > /dev/null 2>&1; then
            mc mb "$ALIAS/$BUCKET"
            echo "Bucket $BUCKET created."
        else
            echo "Bucket $BUCKET already exists."
        fi

        # Apply Policy
        # Check if bucket name implies public access or if generic policy needed
        # We can implement a simple rule: if bucket name contains "public", set public download
        if [[ "$BUCKET" == *"public"* ]]; then
            echo "Setting public download policy for $BUCKET"
            mc anonymous set download "$ALIAS/$BUCKET"
        fi

        # If policy.json exists in mounted dir, we could try to apply it
        # But mc policy set-json is the command.
        POLICY_FILE="/s3-init/policy.json"
        if [ -f "$POLICY_FILE" ]; then
             echo "Applying custom policy from $POLICY_FILE to $BUCKET"
             mc policy set-json "$POLICY_FILE" "$ALIAS/$BUCKET"
        fi
    done
fi

echo "S3 Setup Complete."
