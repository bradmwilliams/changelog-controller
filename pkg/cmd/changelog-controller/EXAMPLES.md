# Changelog Controller - Configuration Examples

Quick reference for common configuration scenarios.

## Development

### Minimal (Filesystem Only)
```bash
# Uses default filesystem storage and HTTP port 8080
./changelog-controller start
```

### With Custom Directory and Port
```bash
./changelog-controller start \
  --http-port=8080 \
  --filesystem-base-dir=/tmp/my-changelogs
```

### Custom HTTP Port
```bash
# Run on a different port
./changelog-controller start --http-port=9090

# Or use environment variable
export HTTP_PORT=9090
./changelog-controller start
```

---

## Single Instance Deployment

### SQLite + Filesystem Backup
```bash
./changelog-controller start \
  --sqlite-enabled=true \
  --sqlite-database-path=/var/lib/changelogs.db \
  --filesystem-enabled=true \
  --filesystem-base-dir=/var/lib/changelog-storage
```

**Environment Variable Version:**
```bash
export SQLITE_ENABLED=true
export SQLITE_DATABASE_PATH=/var/lib/changelogs.db
export FILESYSTEM_ENABLED=true
export FILESYSTEM_BASE_DIR=/var/lib/changelog-storage

./changelog-controller start
```

---

## Production - BigQuery

### BigQuery Only
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/etc/gcp/service-account.json
export BIGQUERY_ENABLED=true
export BIGQUERY_PROJECT_ID=production-project
export BIGQUERY_DATASET_ID=changelogs
export BIGQUERY_TABLE_ID=release_changelogs
export FILESYSTEM_ENABLED=false

./changelog-controller start
```

### BigQuery + Local Backup
```bash
./changelog-controller start \
  --bigquery-enabled=true \
  --bigquery-project-id=production-project \
  --bigquery-dataset-id=changelogs \
  --bigquery-table-id=release_changelogs \
  --filesystem-enabled=true \
  --filesystem-base-dir=/mnt/backup/changelogs
```

---

## Production - Firestore

### Firestore Only
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/etc/gcp/service-account.json
export FIRESTORE_ENABLED=true
export FIRESTORE_PROJECT_ID=production-project
export FIRESTORE_COLLECTION_ID=changelogs
export FILESYSTEM_ENABLED=false

./changelog-controller start
```

### Firestore + Local Backup
```bash
./changelog-controller start \
  --firestore-enabled=true \
  --firestore-project-id=production-project \
  --firestore-collection-id=changelogs \
  --filesystem-enabled=true \
  --filesystem-base-dir=/mnt/backup/changelogs
```

---

## High Availability

### All Backends (Maximum Redundancy)
```bash
export BIGQUERY_ENABLED=true
export BIGQUERY_PROJECT_ID=prod-project
export BIGQUERY_DATASET_ID=changelogs

export SQLITE_ENABLED=true
export SQLITE_DATABASE_PATH=/var/lib/changelogs.db

export FILESYSTEM_ENABLED=true
export FILESYSTEM_BASE_DIR=/var/lib/changelog-storage

./changelog-controller start
```

---

## Kubernetes

### Development (ConfigMap)
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: changelog-controller-config
  namespace: openshift-release
data:
  FILESYSTEM_ENABLED: "true"
  FILESYSTEM_BASE_DIR: "/data/changelogs"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: changelog-controller
  namespace: openshift-release
spec:
  replicas: 1
  selector:
    matchLabels:
      app: changelog-controller
  template:
    metadata:
      labels:
        app: changelog-controller
    spec:
      containers:
      - name: controller
        image: changelog-controller:latest
        command: ["/changelog-controller", "start"]
        envFrom:
        - configMapRef:
            name: changelog-controller-config
        volumeMounts:
        - name: data
          mountPath: /data/changelogs
      volumes:
      - name: data
        emptyDir: {}
```

### Production (BigQuery + PVC)
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gcp-credentials
  namespace: openshift-release
type: Opaque
data:
  key.json: <base64-encoded-service-account-key>
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: changelog-controller-config
  namespace: openshift-release
data:
  BIGQUERY_ENABLED: "true"
  BIGQUERY_PROJECT_ID: "openshift-release-prod"
  BIGQUERY_DATASET_ID: "changelogs"
  BIGQUERY_TABLE_ID: "release_changelogs"
  FILESYSTEM_ENABLED: "true"
  FILESYSTEM_BASE_DIR: "/data/changelogs"
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: changelog-storage
  namespace: openshift-release
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: changelog-controller
  namespace: openshift-release
spec:
  replicas: 1
  selector:
    matchLabels:
      app: changelog-controller
  template:
    metadata:
      labels:
        app: changelog-controller
    spec:
      serviceAccountName: changelog-controller
      containers:
      - name: controller
        image: changelog-controller:latest
        command: ["/changelog-controller", "start"]
        envFrom:
        - configMapRef:
            name: changelog-controller-config
        env:
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /var/secrets/google/key.json
        volumeMounts:
        - name: data
          mountPath: /data/changelogs
        - name: gcp-credentials
          mountPath: /var/secrets/google
          readOnly: true
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: changelog-storage
      - name: gcp-credentials
        secret:
          secretName: gcp-credentials
```

---

## Docker

### Docker Run with Environment Variables
```bash
docker run -d \
  --name changelog-controller \
  -e BIGQUERY_ENABLED=true \
  -e BIGQUERY_PROJECT_ID=my-project \
  -e BIGQUERY_DATASET_ID=changelogs \
  -e FILESYSTEM_ENABLED=true \
  -e FILESYSTEM_BASE_DIR=/data \
  -v /var/lib/changelogs:/data \
  -v ~/.config/gcloud:/root/.config/gcloud:ro \
  -p 8443:8443 \
  changelog-controller:latest start
```

### Docker Compose
```yaml
version: '3.8'
services:
  changelog-controller:
    image: changelog-controller:latest
    command: start
    ports:
      - "8443:8443"
    environment:
      BIGQUERY_ENABLED: "true"
      BIGQUERY_PROJECT_ID: "my-project"
      BIGQUERY_DATASET_ID: "changelogs"
      BIGQUERY_TABLE_ID: "release_changelogs"
      SQLITE_ENABLED: "true"
      SQLITE_DATABASE_PATH: "/data/changelogs.db"
      FILESYSTEM_ENABLED: "true"
      FILESYSTEM_BASE_DIR: "/data/changelog-storage"
      GOOGLE_APPLICATION_CREDENTIALS: "/secrets/gcp-key.json"
    volumes:
      - changelog-data:/data
      - ./gcp-credentials.json:/secrets/gcp-key.json:ro
    restart: unless-stopped

volumes:
  changelog-data:
```

---

## Testing Storage Configuration

### Test Filesystem Storage
```bash
# Start controller
./changelog-controller start \
  --filesystem-enabled=true \
  --filesystem-base-dir=/tmp/test-changelogs &

# Wait for startup
sleep 5

# Test the endpoint
curl "http://localhost:8443/changelog?from=1.0.0&to=2.0.0&arch=amd64"

# Check storage
ls -la /tmp/test-changelogs/

# Cleanup
pkill -f changelog-controller
rm -rf /tmp/test-changelogs
```

### Test SQLite Storage
```bash
# Start controller
./changelog-controller start \
  --sqlite-enabled=true \
  --sqlite-database-path=/tmp/test.db \
  --filesystem-enabled=false &

# Wait for startup
sleep 5

# Generate changelog
curl "http://localhost:8443/changelog?from=1.0.0&to=2.0.0&arch=amd64"

# Check database
sqlite3 /tmp/test.db "SELECT * FROM changelogs;"

# Cleanup
pkill -f changelog-controller
rm -f /tmp/test.db
```

### Test BigQuery Storage (requires GCP credentials)
```bash
# Set up credentials
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# Start controller
./changelog-controller start \
  --bigquery-enabled=true \
  --bigquery-project-id=test-project \
  --bigquery-dataset-id=test_changelogs \
  --bigquery-table-id=test_releases \
  --filesystem-enabled=false &

# Monitor logs
tail -f /var/log/changelog-controller.log

# Test endpoint
curl "http://localhost:8443/changelog?from=4.15.0&to=4.16.0&arch=amd64"

# Query BigQuery
bq query --use_legacy_sql=false \
  'SELECT * FROM test_changelogs.test_releases LIMIT 10'

# Cleanup
pkill -f changelog-controller
```

### Test Firestore Storage (requires GCP credentials)
```bash
# Set up credentials
gcloud auth application-default login

# Start controller
./changelog-controller start \
  --firestore-enabled=true \
  --firestore-project-id=openshift-crt \
  --firestore-database-id=changelog-controller \
  --firestore-collection-id=changelogs \
  --filesystem-enabled=false &

# Wait for startup
sleep 5

# Test endpoint
curl "http://localhost:8080/changelog?from=4.15.0&to=4.16.0&arch=amd64"

# Cleanup
pkill -f changelog-controller
```

---

## Troubleshooting Commands

### Check Storage Backend Status
```bash
# View logs for storage initialization
kubectl logs -l app=changelog-controller | grep -i "storage"

# Expected output:
# I0414 12:00:00 cmd.go:151] Initializing BigQuery storage...
# I0414 12:00:00 cmd.go:167] BigQuery storage initialized successfully
# I0414 12:00:00 cmd.go:215] Initialized 3 storage backend(s)
```

### Verify Filesystem Storage
```bash
# Check directory exists
kubectl exec -it changelog-controller-xxx -- ls -la /var/lib/changelog-storage/

# Check file count
kubectl exec -it changelog-controller-xxx -- \
  find /var/lib/changelog-storage -type f -name "*.txt" | wc -l
```

### Verify SQLite Storage
```bash
# Check database file
kubectl exec -it changelog-controller-xxx -- \
  sqlite3 /var/lib/changelogs.db "SELECT COUNT(*) FROM changelogs;"

# View recent entries
kubectl exec -it changelog-controller-xxx -- \
  sqlite3 /var/lib/changelogs.db \
  "SELECT from_version, to_version, created_at FROM changelogs ORDER BY created_at DESC LIMIT 5;"
```

### Verify BigQuery Storage
```bash
# Check table exists
bq show my-project:changelogs.release_changelogs

# Count rows
bq query --use_legacy_sql=false \
  'SELECT COUNT(*) FROM changelogs.release_changelogs'

# View recent entries
bq query --use_legacy_sql=false \
  'SELECT from_version, to_version, created_at 
   FROM changelogs.release_changelogs 
   ORDER BY created_at DESC 
   LIMIT 5'
```

---

## Migration Examples

### Migrate from Filesystem to BigQuery
```bash
# Step 1: Enable both backends
export FILESYSTEM_ENABLED=true
export FILESYSTEM_BASE_DIR=/var/lib/changelog-storage
export BIGQUERY_ENABLED=true
export BIGQUERY_PROJECT_ID=my-project

./changelog-controller start

# Step 2: New changelogs will be written to both
# Old changelogs remain in filesystem only

# Step 3: Optional - backfill BigQuery from filesystem
# (requires custom migration script)

# Step 4: Once backfilled, disable filesystem
export FILESYSTEM_ENABLED=false
```

### Backup SQLite to Filesystem
```bash
# Run with both enabled
./changelog-controller start \
  --sqlite-enabled=true \
  --sqlite-database-path=/var/lib/changelogs.db \
  --filesystem-enabled=true \
  --filesystem-base-dir=/backup/changelogs

# Both backends will stay in sync
```

---

## API Usage Examples

### Querying Changelogs with PullSpec Information

The `/changelog` endpoint automatically enriches responses with container image pull specifications when available.

#### JSON Response Format

**Request:**
```bash
curl "http://localhost:8080/changelog?from=4.21.9&to=4.21.10&arch=amd64&format=json"
```

**Response:**
```json
{
  "from": "4.21.9",
  "to": "4.21.10",
  "fromPullSpec": "quay.io/openshift-release-dev/ocp-release:4.21.9-x86_64",
  "toPullSpec": "quay.io/openshift-release-dev/ocp-release:4.21.10-x86_64",
  "changelog": "# Release 4.21.10\n\n## Changes\n- Bug fixes..."
}
```

The `fromPullSpec` and `toPullSpec` fields are automatically populated by querying the OpenShift release-controller endpoints. These fields:
- Are optional and only included when the PullSpecs can be resolved
- Contain the exact container image reference needed to pull the release
- Are architecture-specific (e.g., `-x86_64`, `-aarch64`, `-s390x`, etc.)

#### Markdown Response Format

**Request:**
```bash
curl "http://localhost:8080/changelog?from=4.21.9&to=4.21.10&arch=amd64&format=md"
```

**Response:**
```markdown
<!-- PullSpec Information -->
<!-- From: quay.io/openshift-release-dev/ocp-release:4.21.9-x86_64 -->
<!-- To: quay.io/openshift-release-dev/ocp-release:4.21.10-x86_64 -->

# Release 4.21.10

## Changes
- Bug fixes...
```

For markdown format, PullSpec information is included as HTML comments at the top of the response.

#### Error Handling

If PullSpecs cannot be resolved (e.g., release not found, release-controller unavailable), the endpoint will still return the changelog without the PullSpec fields:

```json
{
  "from": "4.21.9",
  "to": "4.21.10",
  "changelog": "# Release 4.21.10..."
}
```

This ensures backward compatibility and graceful degradation when the release-controller is unavailable.


---

## HTTP Server Architecture

The changelog controller runs **two separate HTTP servers**:

### 1. Controller Server (Default Port: varies by deployment)
- Provided by the Kubernetes controller framework
- Endpoints:
  - `/metrics` - Prometheus metrics
  - `/readyz` - Readiness probe
  - `/healthz` - Health check
  - `/livez` - Liveness probe
- Configured via controller framework flags

### 2. Changelog HTTP Server (Default Port: 8080)
- Dedicated server for changelog API endpoints
- Configurable via `--http-port` flag or `HTTP_PORT` environment variable
- Endpoints:
  - `/` - API documentation
  - `/changelog` - Generate changelogs
  - `/status` - Cache and storage status

**Example Configuration:**
```bash
# Controller metrics on default port (e.g., 8443)
# Changelog API on port 8080
./changelog-controller start --http-port=8080

# Access changelog API
curl http://localhost:8080/changelog?from=4.21.9&to=4.21.10

# Access controller metrics
curl https://localhost:8443/metrics
```

**Port Configuration Examples:**
```bash
# Default (port 8080)
./changelog-controller start

# Custom port via flag
./changelog-controller start --http-port=9090

# Custom port via environment variable
HTTP_PORT=9090 ./changelog-controller start

# In Kubernetes deployment
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: changelog-controller
    env:
    - name: HTTP_PORT
      value: "8080"
    ports:
    - containerPort: 8080
      name: changelog-http
      protocol: TCP
```

**Why Separate Servers?**
- **Separation of Concerns**: Controller operations (metrics, health checks) separate from changelog API
- **Independent Scaling**: Can configure different resource limits or network policies
- **Flexible Deployment**: Can expose changelog API publicly while keeping controller endpoints internal
- **Custom Configuration**: Different ports, TLS settings, or access controls per server

