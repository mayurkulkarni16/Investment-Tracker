package services

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const MaxDocumentBytes = 10 << 20

var documentTypes = map[string]string{
	".pdf": "application/pdf", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".webp": "image/webp", ".heic": "image/heic", ".txt": "text/plain", ".csv": "text/csv",
	".doc": "application/msword", ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls": "application/vnd.ms-excel", ".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
}

var documentCategories = map[string]bool{"agreement": true, "receipt": true, "policy": true, "statement": true, "tax": true, "id_proof": true, "other": true}

// DocumentService stores attachments (agreements, receipts, policies) in GridFS.
type DocumentService struct {
	repo   *repository.DocumentRepo
	bucket *gridfs.Bucket
}

func NewDocumentService(db *mongo.Database, repo *repository.DocumentRepo) (*DocumentService, error) {
	bucket, err := gridfs.NewBucket(db, options.GridFSBucket().SetName("documents"))
	if err != nil {
		return nil, err
	}
	return &DocumentService{repo: repo, bucket: bucket}, nil
}

type DocumentUpload struct {
	Title, FileName, Category, AssetType, AssetID, ExpiryDate, Notes string
	Size                                                             int64
	Reader                                                           io.Reader
}

func (s *DocumentService) Upload(ctx context.Context, userID string, up DocumentUpload) (*models.Document, error) {
	if up.Size <= 0 {
		return nil, invalidf("file is empty")
	}
	if up.Size > MaxDocumentBytes {
		return nil, invalidf("file is larger than 10 MB")
	}
	name := filepath.Base(strings.ReplaceAll(up.FileName, "\\", "/"))
	contentType, ok := documentTypes[strings.ToLower(filepath.Ext(name))]
	if !ok {
		return nil, invalidf("unsupported file type; allowed: pdf, images, txt, csv, doc(x), xls(x)")
	}
	category := strings.ToLower(strings.TrimSpace(up.Category))
	if category == "" {
		category = "other"
	}
	if !documentCategories[category] {
		return nil, invalidf("invalid category")
	}
	if up.AssetType != "" && !assignableTypes[up.AssetType] {
		return nil, invalidf("unsupported asset_type %q", up.AssetType)
	}
	if up.AssetID != "" {
		if _, err := primitive.ObjectIDFromHex(up.AssetID); err != nil {
			return nil, invalidf("invalid asset_id")
		}
	}
	var expiry *time.Time
	if up.ExpiryDate != "" {
		t, err := parseDate(up.ExpiryDate)
		if err != nil {
			return nil, err
		}
		expiry = &t
	}
	title := strings.TrimSpace(up.Title)
	if title == "" {
		title = strings.TrimSuffix(name, filepath.Ext(name))
	}

	fileID, err := s.bucket.UploadFromStream(name, io.LimitReader(up.Reader, MaxDocumentBytes+1))
	if err != nil {
		return nil, err
	}
	doc := &models.Document{
		UserID: userID, FileID: fileID, Title: title, FileName: name, ContentType: contentType, Size: up.Size,
		Category: category, AssetType: up.AssetType, AssetID: up.AssetID, ExpiryDate: expiry,
		Notes: up.Notes, CreatedAt: time.Now(),
	}
	id, err := s.repo.Insert(ctx, doc)
	if err != nil {
		_ = s.bucket.Delete(fileID)
		return nil, err
	}
	doc.ID = id
	return doc, nil
}

func (s *DocumentService) List(ctx context.Context, userID, assetType, assetID string) ([]models.Document, error) {
	filter := bson.M{}
	if assetType != "" {
		filter["asset_type"] = assetType
	}
	if assetID != "" {
		filter["asset_id"] = assetID
	}
	docs, err := s.repo.Find(ctx, userID, filter)
	if err != nil {
		return nil, err
	}
	if docs == nil {
		docs = []models.Document{}
	}
	return docs, nil
}

// Open returns the metadata and a reader for the stored file.
func (s *DocumentService) Open(ctx context.Context, id string) (*models.Document, io.ReadCloser, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, nil, err
	}
	doc, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return nil, nil, err
	}
	stream, err := s.bucket.OpenDownloadStream(doc.FileID)
	if err != nil {
		return nil, nil, notFoundf("file content is missing")
	}
	return doc, stream, nil
}

func (s *DocumentService) Delete(ctx context.Context, id string) error {
	oid, err := parseObjectID(id)
	if err != nil {
		return err
	}
	doc, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, oid); err != nil {
		return err
	}
	if err := s.bucket.Delete(doc.FileID); err != nil && err != gridfs.ErrFileNotFound {
		return err
	}
	return nil
}
