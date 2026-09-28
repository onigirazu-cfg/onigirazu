package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config is the managed_state section of onigirazu.yml for a shared
// state in an S3 bucket (AWS, Garage, MinIO, ...). Credentials come from
// AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY (or AWS_PROFILE and
// ~/.aws/credentials), never from the file.
type S3Config struct {
	Bucket string `yaml:"bucket" json:"bucket"`
	// Prefix goes before <playbook>.state.json, e.g. "prod/"
	Prefix string `yaml:"prefix" json:"prefix"`
	// Endpoint is host[:port]; empty is AWS
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	Region   string `yaml:"region" json:"region"`
	// Insecure talks plain HTTP (a test MinIO)
	Insecure bool `yaml:"insecure" json:"insecure"`
	// PathStyle forces bucket-in-path URLs (most self-hosted servers)
	PathStyle bool `yaml:"path_style" json:"path_style"`
}

// S3Store keeps the state of one playbook as an object. The lock needs no
// conditional writes (Garage ignores If-None-Match): a run puts its own
// object under <state>.lock/ and holds the lock only when a listing right
// after shows no other; otherwise it takes its object back and waits. With
// read-after-write consistent puts and listings two runs cannot both see
// themselves alone.
type S3Store struct {
	client     *minio.Client
	bucket     string
	key        string
	lockPrefix string
}

// NewS3Store is the S3 store of a playbook
func NewS3Store(cfg S3Config, playbook string) (*S3Store, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("managed_state: bucket is required for the s3 backend")
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "s3.amazonaws.com"
	}
	lookup := minio.BucketLookupAuto
	if cfg.PathStyle {
		lookup = minio.BucketLookupPath
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewChainCredentials([]credentials.Provider{
			&credentials.EnvAWS{}, &credentials.FileAWSCredentials{}, &credentials.IAM{},
		}),
		Secure:       !cfg.Insecure,
		Region:       cfg.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("managed_state: %w", err)
	}
	base := strings.TrimSuffix(filepath.Base(playbook), filepath.Ext(playbook))
	key := strings.TrimPrefix(cfg.Prefix, "/") + base + ".state.json"
	return &S3Store{client: client, bucket: cfg.Bucket, key: key, lockPrefix: key + ".lock/"}, nil
}

func (s *S3Store) String() string { return "s3://" + s.bucket + "/" + s.key }

func notFound(err error) bool {
	return minio.ToErrorResponse(err).StatusCode == http.StatusNotFound
}

func (s *S3Store) get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

// Load implements Store
func (s *S3Store) Load(ctx context.Context) (*State, error) {
	data, err := s.get(ctx, s.key)
	if notFound(err) {
		return &State{Version: Version}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("state %s: %w", s, err)
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("state %s: %w", s, err)
	}
	if st.Version > Version {
		return nil, fmt.Errorf("state %s has version %d, this onigirazu reads up to %d", s, st.Version, Version)
	}
	return &st, nil
}

// Save implements Store
func (s *S3Store) Save(ctx context.Context, st *State) error {
	st.Version = Version
	st.sort()
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	_, err = s.client.PutObject(ctx, s.bucket, s.key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: "application/json"})
	if err != nil {
		return fmt.Errorf("state %s: %w", s, err)
	}
	return nil
}

type s3Lock struct {
	store *S3Store
	id    string
}

// Lock implements Store
func (s *S3Store) Lock(ctx context.Context, operation string, timeout time.Duration) (Unlocker, error) {
	info := LockInfo{ID: newLockID(), Who: whoAmI(), Operation: operation, Created: time.Now().UTC()}
	data, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	mine := s.lockPrefix + info.ID
	deadline := time.Now().Add(timeout)
	for {
		if _, err := s.client.PutObject(ctx, s.bucket, mine, bytes.NewReader(data), int64(len(data)),
			minio.PutObjectOptions{ContentType: "application/json"}); err != nil {
			return nil, fmt.Errorf("lock %s: %w", s, err)
		}
		others, err := s.lockObjects(ctx)
		if err != nil {
			_ = s.client.RemoveObject(ctx, s.bucket, mine, minio.RemoveObjectOptions{})
			return nil, fmt.Errorf("lock %s: %w", s, err)
		}
		delete(others, info.ID)
		if len(others) == 0 {
			return &s3Lock{store: s, id: info.ID}, nil
		}
		if err := s.client.RemoveObject(ctx, s.bucket, mine, minio.RemoveObjectOptions{}); err != nil {
			return nil, fmt.Errorf("lock %s: %w", s, err)
		}
		if time.Now().After(deadline) {
			return nil, &LockedError{Path: s.String(), Info: s.holder(ctx, others)}
		}
		// two runs that met back off for different times
		wait := 500*time.Millisecond + time.Duration(randomInt(1000))*time.Millisecond
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// lockObjects lists the lock objects by ID
func (s *S3Store) lockObjects(ctx context.Context) (map[string]bool, error) {
	ids := map[string]bool{}
	for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: s.lockPrefix}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		ids[strings.TrimPrefix(obj.Key, s.lockPrefix)] = true
	}
	return ids, nil
}

// holder describes the oldest other lock
func (s *S3Store) holder(ctx context.Context, ids map[string]bool) LockInfo {
	var oldest LockInfo
	for id := range ids {
		info := LockInfo{ID: id, Who: "unknown"}
		if data, err := s.get(ctx, s.lockPrefix+id); err == nil {
			_ = json.Unmarshal(data, &info)
		}
		if oldest.ID == "" || info.Created.Before(oldest.Created) {
			oldest = info
		}
	}
	return oldest
}

// ForceUnlock implements Store
func (s *S3Store) ForceUnlock(ctx context.Context, id string) error {
	ids, err := s.lockObjects(ctx)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("managed state %s is not locked", s)
	}
	if !ids[id] {
		return fmt.Errorf("lock ID %s does not match the lock of %s", id, s)
	}
	return s.client.RemoveObject(ctx, s.bucket, s.lockPrefix+id, minio.RemoveObjectOptions{})
}

// Release removes the lock if it is still this run's
func (l *s3Lock) Release() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return l.store.ForceUnlock(ctx, l.id)
}
