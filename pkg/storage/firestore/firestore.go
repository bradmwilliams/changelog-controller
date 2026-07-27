package firestore

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/bradmwilliams/changelog-controller/pkg/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FirestoreStorage implements changelog storage using Google Cloud Firestore
type FirestoreStorage struct {
	projectID    string
	collectionID string
	client       *firestore.Client
}

// changelogDoc represents a document in the Firestore changelogs collection.
// Changelog is stored gzip-compressed to stay well under Firestore's 1 MiB
// per-document limit.
type changelogDoc struct {
	FromVersion string    `firestore:"fromVersion"`
	ToVersion   string    `firestore:"toVersion"`
	Changelog   []byte    `firestore:"changelog"`
	CreatedAt   time.Time `firestore:"createdAt"`
}

// Config holds configuration for Firestore storage
type Config struct {
	ProjectID    string
	DatabaseID   string
	CollectionID string
}

var _ storage.ChangelogStorage = (*FirestoreStorage)(nil)
var _ storage.ChangelogLister = (*FirestoreStorage)(nil)

// NewFirestoreStorage creates a new Firestore storage backend
func NewFirestoreStorage(ctx context.Context, cfg Config) (*FirestoreStorage, error) {
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("projectID is required")
	}
	if cfg.CollectionID == "" {
		return nil, fmt.Errorf("collectionID is required")
	}

	var client *firestore.Client
	var err error
	if cfg.DatabaseID != "" {
		client, err = firestore.NewClientWithDatabase(ctx, cfg.ProjectID, cfg.DatabaseID)
	} else {
		client, err = firestore.NewClient(ctx, cfg.ProjectID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client: %w", err)
	}

	return &FirestoreStorage{
		projectID:    cfg.ProjectID,
		collectionID: cfg.CollectionID,
		client:       client,
	}, nil
}

// docID generates a deterministic document ID for a given version range
func docID(from, to string) string {
	key := fmt.Sprintf("%s->%s", from, to)
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash[:16])
}

// Get retrieves a changelog from Firestore
func (s *FirestoreStorage) Get(ctx context.Context, from, to string) (string, bool, error) {
	snap, err := s.client.Collection(s.collectionID).Doc(docID(from, to)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to get document from Firestore: %w", err)
	}

	var doc changelogDoc
	if err := snap.DataTo(&doc); err != nil {
		return "", false, fmt.Errorf("failed to decode Firestore document: %w", err)
	}

	changelog, err := decompress(doc.Changelog)
	if err != nil {
		return "", false, fmt.Errorf("failed to decompress changelog: %w", err)
	}

	return changelog, true, nil
}

// Set stores a changelog in Firestore
func (s *FirestoreStorage) Set(ctx context.Context, from, to, changelog string) error {
	compressed, err := compress(changelog)
	if err != nil {
		return fmt.Errorf("failed to compress changelog: %w", err)
	}

	doc := changelogDoc{
		FromVersion: from,
		ToVersion:   to,
		Changelog:   compressed,
		CreatedAt:   time.Now(),
	}

	if _, err := s.client.Collection(s.collectionID).Doc(docID(from, to)).Set(ctx, doc); err != nil {
		return fmt.Errorf("failed to set document in Firestore: %w", err)
	}

	return nil
}

// Close closes the Firestore client
func (s *FirestoreStorage) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// List returns all changelog entries stored in Firestore
// Implements the ChangelogLister interface for cache warming
func (s *FirestoreStorage) List(ctx context.Context) ([]storage.ChangelogEntry, error) {
	it := s.client.Collection(s.collectionID).Documents(ctx)
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

		var doc changelogDoc
		if err := snap.DataTo(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode Firestore document: %w", err)
		}

		changelog, err := decompress(doc.Changelog)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress changelog: %w", err)
		}

		entries = append(entries, storage.ChangelogEntry{
			FromVersion: doc.FromVersion,
			ToVersion:   doc.ToVersion,
			Changelog:   changelog,
		})
	}

	return entries, nil
}

// compress gzip-compresses a changelog string
func compress(changelog string) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte(changelog)); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decompress gzip-decompresses a changelog
func decompress(data []byte) (string, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(decompressed), nil
}
