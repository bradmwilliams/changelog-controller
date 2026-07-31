# Changelog Controller Configuration Guide

This guide explains how to configure the Release Payload Changelog Controller, including storage backends and HTTP server settings.

## Storage Backend Configuration

The controller supports four storage backends that can be enabled individually or in combination:

1. **Filesystem Storage** (enabled by default)
2. **SQLite Storage**
3. **BigQuery Storage**
4. **Firestore Storage**

### Configuration Methods

Each storage backend can be configured via:
- **Command-line flags**: `--storage-option value`
- **Environment variables**: `STORAGE_OPTION=value`

Command-line flags take precedence over environment variables.

---

## Filesystem Storage

**Default**: Enabled

Simple file-based storage ideal for development and small deployments.

### Command-Line Flags

```bash
./changelog-controller start \
  --filesystem-enabled=true \
  --filesystem-base-dir=/var/lib/changelog-storage
```

### Environment Variables

```bash
export FILESYSTEM_ENABLED=true
export FILESYSTEM_BASE_DIR=/var/lib/changelog-storage
./changelog-controller start
```

### Configuration Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--filesystem-enabled` | `FILESYSTEM_ENABLED` | `true` | Enable filesystem storage |
| `--filesystem-base-dir` | `FILESYSTEM_BASE_DIR` | `/var/lib/changelog-storage` | Base directory for storage |

### Notes

- Automatically creates directory structure
- Each changelog stored as hash-based filename
- Includes metadata files for debugging
- No external dependencies

---

## SQLite Storage

**Default**: Disabled

Local database storage ideal for single-instance deployments.

### Command-Line Flags

```bash
./changelog-controller start \
  --sqlite-enabled=true \
  --sqlite-database-path=/var/lib/changelogs.db
```

### Environment Variables

```bash
export SQLITE_ENABLED=true
export SQLITE_DATABASE_PATH=/var/lib/changelogs.db
./changelog-controller start
```

### Configuration Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--sqlite-enabled` | `SQLITE_ENABLED` | `false` | Enable SQLite storage |
| `--sqlite-database-path` | `SQLITE_DATABASE_PATH` | `/var/lib/changelogs.db` | Database file path |

### Notes

- Automatically initializes schema on first run
- Supports UPSERT for updates
- Use `:memory:` for in-memory database (testing only)
- Requires write permissions to database directory

---

## BigQuery Storage

**Default**: Disabled

Cloud-based storage ideal for production deployments with high scalability.

### Command-Line Flags

```bash
./changelog-controller start \
  --bigquery-enabled=true \
  --bigquery-project-id=my-gcp-project \
  --bigquery-dataset-id=changelogs \
  --bigquery-table-id=release_changelogs
```

### Environment Variables

```bash
export BIGQUERY_ENABLED=true
export BIGQUERY_PROJECT_ID=my-gcp-project
export BIGQUERY_DATASET_ID=changelogs
export BIGQUERY_TABLE_ID=release_changelogs
./changelog-controller start
```

### Configuration Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--bigquery-enabled` | `BIGQUERY_ENABLED` | `false` | Enable BigQuery storage |
| `--bigquery-project-id` | `BIGQUERY_PROJECT_ID` | *(required)* | GCP project ID |
| `--bigquery-dataset-id` | `BIGQUERY_DATASET_ID` | `changelogs` | BigQuery dataset ID |
| `--bigquery-table-id` | `BIGQUERY_TABLE_ID` | `release_changelogs` | BigQuery table ID |

### Prerequisites

1. **GCP Authentication**: Set up application default credentials
   ```bash
   gcloud auth application-default login
   # Or use service account
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
   ```

2. **BigQuery Setup**: Create dataset and table
   ```sql
   CREATE DATASET IF NOT EXISTS changelogs;
   
   CREATE TABLE IF NOT EXISTS changelogs.release_changelogs (
       from_version STRING NOT NULL,
       to_version STRING NOT NULL,
       changelog STRING NOT NULL,
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   );
   ```

3. **IAM Permissions**: Ensure service account has:
   - `bigquery.dataEditor` role (or `bigquery.tables.updateData` and `bigquery.tables.getData`)

### Notes

- Requires GCP credentials
- Supports automatic authentication via ADC
- Scalable for production workloads
- Query costs apply based on usage

---

## Firestore Storage

**Default**: Disabled

Cloud-based storage using Firestore as an index and GCS as the content store. Firestore documents hold version metadata and a reference to the GCS object that contains the actual changelog content.

### Command-Line Flags

```bash
./changelog-controller start \
  --firestore-enabled=true \
  --firestore-project-id=my-gcp-project \
  --firestore-collection-id=changelogs \
  --firestore-gcs-bucket=my-changelog-bucket
```

### Environment Variables

```bash
export FIRESTORE_ENABLED=true
export FIRESTORE_PROJECT_ID=my-gcp-project
export FIRESTORE_COLLECTION_ID=changelogs
export FIRESTORE_GCS_BUCKET=my-changelog-bucket
./changelog-controller start
```

### Configuration Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--firestore-enabled` | `FIRESTORE_ENABLED` | `false` | Enable Firestore storage |
| `--firestore-project-id` | `FIRESTORE_PROJECT_ID` | *(required)* | GCP project ID |
| `--firestore-database-id` | `FIRESTORE_DATABASE_ID` | `(default)` | Firestore database ID |
| `--firestore-collection-id` | `FIRESTORE_COLLECTION_ID` | `changelogs` | Firestore collection ID |
| `--firestore-gcs-bucket` | `FIRESTORE_GCS_BUCKET` | *(required)* | GCS bucket for changelog content |

### Prerequisites

1. **GCP Authentication**: Set up application default credentials
   ```bash
   gcloud auth application-default login
   # Or use service account
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
   ```

2. **Firestore Database**: A Firestore database in Native mode must already exist in the project. The database can be created through the [GCP Console](https://console.cloud.google.com/firestore/databases).

3. **GCS Bucket**: A GCS bucket must exist for storing changelog content. The bucket can be public to allow direct consumer access.

### Notes

- Requires GCP credentials for both Firestore and GCS
- Firestore acts as an index; changelog content is stored as GCS objects
- GCS objects are stored under `changelogs/` with hash-based filenames
- Scalable for production workloads

---

## Multi-Backend Configuration

Enable multiple storage backends for redundancy and failover.

### Example: All Backends Enabled

```bash
./changelog-controller start \
  --bigquery-enabled=true \
  --bigquery-project-id=my-project \
  --sqlite-enabled=true \
  --sqlite-database-path=/var/lib/changelogs.db \
  --filesystem-enabled=true \
  --filesystem-base-dir=/var/lib/changelog-storage
```

### Behavior

When multiple backends are enabled:

1. **Read Operations** (Cache Miss):
   - Query backends in order: Memory → BigQuery → SQLite → Filesystem
   - First successful response is returned and cached
   - Failed backends are logged but don't block

2. **Write Operations** (New Changelog):
   - Write to **all** enabled backends concurrently
   - Failed writes are logged but don't prevent success
   - In-memory cache is always updated

### Recommended Configurations

#### Development
```bash
# Filesystem only (default)
./changelog-controller start
```

#### Single Instance Production
```bash
# SQLite + Filesystem backup
export SQLITE_ENABLED=true
export FILESYSTEM_ENABLED=true
./changelog-controller start
```

#### Multi-Instance Production
```bash
# BigQuery primary, Filesystem backup
export BIGQUERY_ENABLED=true
export BIGQUERY_PROJECT_ID=prod-project
export FILESYSTEM_ENABLED=true
./changelog-controller start
```

#### High Availability
```bash
# All backends for maximum redundancy
export BIGQUERY_ENABLED=true
export BIGQUERY_PROJECT_ID=prod-project
export SQLITE_ENABLED=true
export FILESYSTEM_ENABLED=true
./changelog-controller start
```

---

## Kubernetes Deployment Examples

### ConfigMap for Environment Variables

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: changelog-controller-config
data:
  BIGQUERY_ENABLED: "true"
  BIGQUERY_PROJECT_ID: "my-gcp-project"
  BIGQUERY_DATASET_ID: "changelogs"
  BIGQUERY_TABLE_ID: "release_changelogs"
  SQLITE_ENABLED: "false"
  FILESYSTEM_ENABLED: "true"
  FILESYSTEM_BASE_DIR: "/var/lib/changelog-storage"
```

### Deployment with Storage

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: changelog-controller
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: controller
        image: changelog-controller:latest
        command:
        - /changelog-controller
        - start
        envFrom:
        - configMapRef:
            name: changelog-controller-config
        volumeMounts:
        - name: storage
          mountPath: /var/lib/changelog-storage
        - name: gcp-credentials
          mountPath: /var/secrets/google
        env:
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /var/secrets/google/key.json
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: changelog-storage
      - name: gcp-credentials
        secret:
          secretName: gcp-service-account
```

---

## Validation

The controller validates configuration on startup:

- **BigQuery**: Requires `project-id` when enabled
- **SQLite**: Requires `database-path` when enabled  
- **Firestore**: Requires `project-id` and `collection-id` when enabled
- **Filesystem**: Requires `base-dir` when enabled

Invalid configuration will prevent startup with a descriptive error message.

---

## Monitoring

### Logs

Storage backend initialization is logged on startup:

```
I0414 12:00:00.000000       1 cmd.go:151] Initializing BigQuery storage (project=my-project, dataset=changelogs, table=release_changelogs)
I0414 12:00:00.100000       1 cmd.go:167] BigQuery storage initialized successfully
I0414 12:00:00.101000       1 cmd.go:171] Initializing SQLite storage (path=/var/lib/changelogs.db)
I0414 12:00:00.150000       1 cmd.go:187] SQLite storage initialized successfully
I0414 12:00:00.151000       1 cmd.go:191] Initializing Filesystem storage (base-dir=/var/lib/changelog-storage)
I0414 12:00:00.155000       1 cmd.go:207] Filesystem storage initialized successfully
I0414 12:00:00.156000       1 cmd.go:215] Initialized 3 storage backend(s)
```

Failed backend initialization is logged but doesn't prevent startup:

```
W0414 12:00:00.100000       1 cmd.go:165] Failed to initialize BigQuery storage: project not found
W0414 12:00:00.156000       1 cmd.go:213] No storage backends enabled - changelogs will only be stored in memory cache
```

### Health Checks

The `/readyz` endpoint returns `200 OK` when the controller is ready, regardless of storage backend status.

---

## Troubleshooting

### BigQuery Connection Issues

**Error**: `Failed to initialize BigQuery storage: could not find default credentials`

**Solution**: Set up GCP authentication
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
```

### Firestore Connection Issues

**Error**: `Failed to initialize Firestore storage: ... PermissionDenied`

**Solution**: Ensure you are authenticated via `gcloud auth application-default login` and have access to the target GCP project.

### SQLite Permission Issues

**Error**: `Failed to initialize SQLite storage: unable to open database file`

**Solution**: Ensure directory exists and is writable
```bash
mkdir -p /var/lib
chmod 755 /var/lib
```

### Filesystem Directory Issues

**Error**: `Failed to initialize Filesystem storage: permission denied`

**Solution**: Create directory with proper permissions
```bash
mkdir -p /var/lib/changelog-storage
chmod 755 /var/lib/changelog-storage
```

### No Storage Backends Enabled

**Warning**: `No storage backends enabled - changelogs will only be stored in memory cache`

**Impact**: Changelogs lost on restart

**Solution**: Enable at least one storage backend:
```bash
export FILESYSTEM_ENABLED=true
./changelog-controller start
```

---

## Performance Considerations

### Cache Hierarchy

1. **In-Memory Cache**: Sub-millisecond latency
2. **Storage Backends**: Millisecond to second latency depending on backend

### Backend Characteristics

| Backend | Read Latency | Write Latency | Scalability | Durability |
|---------|--------------|---------------|-------------|------------|
| Filesystem | ~1ms | ~1ms | Single node | Local disk |
| SQLite | ~1-5ms | ~1-10ms | Single node | Local disk |
| BigQuery | ~100-500ms | ~100-500ms | Unlimited | Cloud-grade |
| Firestore | ~10-50ms | ~10-50ms | Unlimited | Cloud-grade |

### Recommendations

- **Development**: Filesystem only (fastest, simplest)
- **Single Instance**: SQLite + Filesystem (good balance)
- **Multi Instance**: Firestore or BigQuery + Filesystem (scalable, redundant)
- **High Traffic**: Firestore or BigQuery only (no local I/O bottlenecks)
