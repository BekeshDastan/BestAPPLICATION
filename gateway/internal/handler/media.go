package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MediaHandler struct {
	internal   *minio.Client // gateway → MinIO (internal docker hostname)
	bucket     string
	publicHost string
	useSSL     bool
}

func NewMediaHandler(endpoint, accessKey, secretKey, bucket, publicHost string, useSSL bool) *MediaHandler {
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		// Force region so the SDK doesn't probe BucketLocation at presign time.
		Region: "us-east-1",
	}
	internal, err := minio.New(endpoint, opts)
	if err != nil {
		slog.Error("minio internal client", "err", err)
	}

	h := &MediaHandler{
		internal:   internal,
		bucket:     bucket,
		publicHost: publicHost,
		useSSL:     useSSL,
	}
	h.ensureBucket()
	return h
}

// ensureBucket creates the bucket and sets a public-read policy if needed.
func (h *MediaHandler) ensureBucket() {
	if h.internal == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := h.internal.BucketExists(ctx, h.bucket)
	if err != nil {
		slog.Warn("minio bucket check failed", "err", err)
		return
	}
	if !exists {
		if err := h.internal.MakeBucket(ctx, h.bucket, minio.MakeBucketOptions{Region: "us-east-1"}); err != nil {
			slog.Warn("minio make bucket failed", "err", err)
			return
		}
	}
	// Set public-read policy so images are viewable without auth
	policy := fmt.Sprintf(`{
		"Version":"2012-10-17",
		"Statement":[{
			"Effect":"Allow",
			"Principal":{"AWS":["*"]},
			"Action":["s3:GetObject"],
			"Resource":["arn:aws:s3:::%s/*"]
		}]
	}`, h.bucket)
	if err := h.internal.SetBucketPolicy(ctx, h.bucket, policy); err != nil {
		slog.Warn("minio set policy failed", "err", err)
	}
}

// GET /media/upload-url — returns a presigned PUT URL + public media URL.
// We sign with the internal client (it can reach MinIO inside the docker
// network) and then rewrite the Host so the browser hits the public address.
func (h *MediaHandler) UploadURL(c *gin.Context) {
	if h.internal == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "media service unavailable"})
		return
	}

	mediaType := c.DefaultQuery("type", "post")
	filename := c.DefaultQuery("filename", "file")
	contentType := c.DefaultQuery("content_type", "application/octet-stream")

	objectName := fmt.Sprintf("%s/%s/%s", mediaType, uuid.New().String(), filename)

	presignedURL, err := h.internal.PresignedPutObject(
		c.Request.Context(),
		h.bucket,
		objectName,
		30*time.Minute,
	)
	if err != nil {
		slog.Error("presign put object failed",
			"err", err,
			"bucket", h.bucket,
			"object", objectName,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate upload URL"})
		return
	}

	// Rewrite host/scheme for the browser.
	presignedURL.Host = h.publicHost
	if h.useSSL {
		presignedURL.Scheme = "https"
	} else {
		presignedURL.Scheme = "http"
	}

	scheme := "http"
	if h.useSSL {
		scheme = "https"
	}
	mediaURL := fmt.Sprintf("%s://%s/%s/%s", scheme, h.publicHost, h.bucket, objectName)

	c.JSON(http.StatusOK, gin.H{
		"upload_url":   presignedURL.String(),
		"media_url":    mediaURL,
		"content_type": contentType,
	})
}
