package firestore

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/firestore"
	gcs "cloud.google.com/go/storage"
	"github.com/bradmwilliams/changelog-controller/pkg/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// FirestoreStorage implements changelog storage using Firestore as an index
// and GCS as the content store. Firestore documents hold version metadata and
// a reference to the GCS object that contains the actual changelog.
type FirestoreStorage struct {
	collectionID string
	bucketName   string
	fsClient     *firestore.Client
	gcsClient    *gcs.Client
}

// indexDoc represents a Firestore document that maps a version pair to a
// GCS object containing the changelog content
type indexDoc struct {
	FromVersion string    `firestore:"fromVersion"`
	ToVersion   string    `firestore:"toVersion"`
	ObjectPath  string    `firestore:"objectPath"`
	CreatedAt   time.Time `firestore:"createdAt"`
}

// Config holds configuration for Firestore + GCS storage
type Config struct {
	ProjectID    string
	DatabaseID   string
	CollectionID string
	BucketName   string
}

var _ storage.ChangelogStorage = (*FirestoreStorage)(nil)
var _ storage.ChangelogLister = (*FirestoreStorage)(nil)

// NewFirestoreStorage creates a new Firestore + GCS storage backend
func NewFirestoreStorage(ctx context.Context, cfg Config) (*FirestoreStorage, error) {
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("projectID is required")
	}
	if cfg.CollectionID == "" {
		return nil, fmt.Errorf("collectionID is required")
	}
	if cfg.BucketName == "" {
		return nil, fmt.Errorf("bucketName is required")
	}

	var fsClient *firestore.Client
	var err error
	if cfg.DatabaseID != "" {
		fsClient, err = firestore.NewClientWithDatabase(ctx, cfg.ProjectID, cfg.DatabaseID)
	} else {
		fsClient, err = firestore.NewClient(ctx, cfg.ProjectID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client: %w", err)
	}

	gcsClient, err := gcs.NewClient(ctx)
	if err != nil {
		fsClient.Close()
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &FirestoreStorage{
		collectionID: cfg.CollectionID,
		bucketName:   cfg.BucketName,
		fsClient:     fsClient,
		gcsClient:    gcsClient,
	}, nil
}

// versionPairHash returns a truncated hex-encoded SHA-256 of the version pair
func versionPairHash(from, to string) string {
	key := fmt.Sprintf("%s->%s", from, to)
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash[:16])
}

// objectPath generates a deterministic GCS object path for a version pair
func objectPath(from, to string) string {
	return fmt.Sprintf("changelogs/%s.json", versionPairHash(from, to))
}

// docID generates a deterministic Firestore document ID for a version pair
func docID(from, to string) string {
	return versionPairHash(from, to)
}

// Get retrieves a changelog by looking up its GCS location in Firestore
// and fetching the content from GCS
func (s *FirestoreStorage) Get(ctx context.Context, from, to string) (string, bool, error) {
	snap, err := s.fsClient.Collection(s.collectionID).Doc(docID(from, to)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to get index from Firestore: %w", err)
	}

	var doc indexDoc
	if err := snap.DataTo(&doc); err != nil {
		return "", false, fmt.Errorf("failed to decode Firestore document: %w", err)
	}

	changelog, err := s.readFromGCS(ctx, doc.ObjectPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to read changelog from GCS: %w", err)
	}

	return changelog, true, nil
}

// Set stores a changelog in GCS and writes the index entry to Firestore
func (s *FirestoreStorage) Set(ctx context.Context, from, to, changelog string) error {
	path := objectPath(from, to)

	if err := s.writeToGCS(ctx, path, changelog); err != nil {
		return fmt.Errorf("failed to write changelog to GCS: %w", err)
	}

	doc := indexDoc{
		FromVersion: from,
		ToVersion:   to,
		ObjectPath:  path,
		CreatedAt:   time.Now(),
	}

	if _, err := s.fsClient.Collection(s.collectionID).Doc(docID(from, to)).Set(ctx, doc); err != nil {
		klog.Warningf("Firestore index write failed after GCS write succeeded (orphaned object: %s): %v", path, err)
		return fmt.Errorf("failed to write index to Firestore: %w", err)
	}

	return nil
}

// Close closes both the Firestore and GCS clients
func (s *FirestoreStorage) Close() error {
	var firstErr error
	if s.fsClient != nil {
		if err := s.fsClient.Close(); err != nil {
			firstErr = err
		}
	}
	if s.gcsClient != nil {
		if err := s.gcsClient.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// List returns all changelog entries by reading the index from Firestore
// and fetching each changelog from GCS
func (s *FirestoreStorage) List(ctx context.Context) ([]storage.ChangelogEntry, error) {
	it := s.fsClient.Collection(s.collectionID).Documents(ctx)
	defer it.Stop()

	var entries []storage.ChangelogEntry
	for {
		snap, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate Firestore documents: %w", err)
		}

		var doc indexDoc
		if err := snap.DataTo(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode Firestore document: %w", err)
		}

		changelog, err := s.readFromGCS(ctx, doc.ObjectPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read changelog from GCS: %w", err)
		}

		entries = append(entries, storage.ChangelogEntry{
			FromVersion: doc.FromVersion,
			ToVersion:   doc.ToVersion,
			Changelog:   changelog,
		})
	}

	return entries, nil
}

func (s *FirestoreStorage) writeToGCS(ctx context.Context, path, content string) error {
	w := s.gcsClient.Bucket(s.bucketName).Object(path).NewWriter(ctx)
	w.ContentType = "application/json"
	_, writeErr := w.Write([]byte(content))
	closeErr := w.Close()
	if writeErr != nil {
		return fmt.Errorf("%s: %w", path, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("%s: %w", path, closeErr)
	}
	return nil
}

func (s *FirestoreStorage) readFromGCS(ctx context.Context, path string) (string, error) {
	r, err := s.gcsClient.Bucket(s.bucketName).Object(path).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	return string(data), nil
}
