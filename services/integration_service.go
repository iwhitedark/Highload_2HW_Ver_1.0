package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"go-microservice/models"
)

// IntegrationService handles S3-compatible storage operations via MinIO
type IntegrationService struct {
	client     *minio.Client
	bucketName string
	mu         sync.RWMutex
	connected  bool
}

var (
	integrationInstance *IntegrationService
	integrationOnce     sync.Once
)

// MinIOConfig holds configuration for MinIO connection
type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	UseSSL          bool
}

// GetDefaultConfig returns default MinIO configuration from environment
func GetDefaultConfig() MinIOConfig {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}

	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}

	bucket := os.Getenv("MINIO_BUCKET")
	if bucket == "" {
		bucket = "users-backup"
	}

	useSSL := os.Getenv("MINIO_USE_SSL") == "true"

	return MinIOConfig{
		Endpoint:        endpoint,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		BucketName:      bucket,
		UseSSL:          useSSL,
	}
}

// GetIntegrationService returns a singleton instance of IntegrationService
func GetIntegrationService() *IntegrationService {
	integrationOnce.Do(func() {
		integrationInstance = &IntegrationService{
			connected: false,
		}
	})
	return integrationInstance
}

// Connect initializes the MinIO client
func (s *IntegrationService) Connect(config MinIOConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to create MinIO client: %w", err)
	}

	s.client = client
	s.bucketName = config.BucketName
	s.connected = true

	// Create bucket if it doesn't exist
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, config.BucketName)
	if err != nil {
		log.Printf("Warning: failed to check bucket existence: %v", err)
		return nil // Don't fail connection, bucket might not be accessible yet
	}

	if !exists {
		err = client.MakeBucket(ctx, config.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Printf("Warning: failed to create bucket: %v", err)
		} else {
			log.Printf("Created bucket: %s", config.BucketName)
		}
	}

	log.Printf("Connected to MinIO at %s", config.Endpoint)
	return nil
}

// IsConnected returns whether the service is connected to MinIO
func (s *IntegrationService) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// BackupUser stores user data in MinIO
func (s *IntegrationService) BackupUser(ctx context.Context, user *models.User) error {
	s.mu.RLock()
	if !s.connected || s.client == nil {
		s.mu.RUnlock()
		return fmt.Errorf("MinIO client not connected")
	}
	client := s.client
	bucket := s.bucketName
	s.mu.RUnlock()

	// Serialize user to JSON
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	objectName := fmt.Sprintf("users/%d.json", user.ID)
	reader := bytes.NewReader(data)

	_, err = client.PutObject(ctx, bucket, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to upload user backup: %w", err)
	}

	log.Printf("User %d backed up to MinIO", user.ID)
	return nil
}

// RestoreUser retrieves user data from MinIO
func (s *IntegrationService) RestoreUser(ctx context.Context, userID int) (*models.User, error) {
	s.mu.RLock()
	if !s.connected || s.client == nil {
		s.mu.RUnlock()
		return nil, fmt.Errorf("MinIO client not connected")
	}
	client := s.client
	bucket := s.bucketName
	s.mu.RUnlock()

	objectName := fmt.Sprintf("users/%d.json", userID)

	obj, err := client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get user backup: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read user backup: %w", err)
	}

	var user models.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

// DeleteUserBackup removes user backup from MinIO
func (s *IntegrationService) DeleteUserBackup(ctx context.Context, userID int) error {
	s.mu.RLock()
	if !s.connected || s.client == nil {
		s.mu.RUnlock()
		return fmt.Errorf("MinIO client not connected")
	}
	client := s.client
	bucket := s.bucketName
	s.mu.RUnlock()

	objectName := fmt.Sprintf("users/%d.json", userID)

	err := client.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete user backup: %w", err)
	}

	log.Printf("User %d backup deleted from MinIO", userID)
	return nil
}

// BackupAllUsers backs up all users to MinIO
func (s *IntegrationService) BackupAllUsers(ctx context.Context, users []*models.User) error {
	for _, user := range users {
		if err := s.BackupUser(ctx, user); err != nil {
			return fmt.Errorf("failed to backup user %d: %w", user.ID, err)
		}
	}
	return nil
}

// ListBackups returns a list of all user backup object names
func (s *IntegrationService) ListBackups(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	if !s.connected || s.client == nil {
		s.mu.RUnlock()
		return nil, fmt.Errorf("MinIO client not connected")
	}
	client := s.client
	bucket := s.bucketName
	s.mu.RUnlock()

	var backups []string
	objectCh := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    "users/",
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}
		backups = append(backups, object.Key)
	}

	return backups, nil
}

// HealthCheck checks MinIO connectivity
func (s *IntegrationService) HealthCheck(ctx context.Context) error {
	s.mu.RLock()
	if !s.connected || s.client == nil {
		s.mu.RUnlock()
		return fmt.Errorf("MinIO client not connected")
	}
	client := s.client
	bucket := s.bucketName
	s.mu.RUnlock()

	_, err := client.BucketExists(ctx, bucket)
	return err
}
