package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPresignStoredObjectKey(t *testing.T) {
	svc, err := NewImageService(&ImageConfig{
		AWSEndpoint: "http://127.0.0.1:1", // presigning is offline; never dialed
		AWSRegion:   "us-east-1", // required: without a region minio dials GetBucketLocation
		AWSAccessID: "testkey", AWSSecret: "testsecret",
		BucketName: "letpai",
	})
	if err != nil {
		t.Fatalf("NewImageService: %v", err)
	}
	ctx := context.Background()

	// Bare object key → presigned URL against bucket+key.
	got := svc.PresignStoredObjectKey(ctx, "payments/abc.png", 15*time.Minute)
	if !strings.Contains(got, "/letpai/payments/abc.png?") || !strings.Contains(got, "X-Amz-Signature") {
		t.Fatalf("expected presigned URL for object key, got %q", got)
	}

	// Full https URL (legacy row) passes through unchanged.
	legacy := "https://s3.rylo.lol/letpai/payments/abc.png"
	if got := svc.PresignStoredObjectKey(ctx, legacy, time.Minute); got != legacy {
		t.Fatalf("legacy URL must pass through, got %q", got)
	}

	// Bare filename without slash (bill-image storage shape) passes through.
	if got := svc.PresignStoredObjectKey(ctx, "abc.png", time.Minute); got != "abc.png" {
		t.Fatalf("bare filename must pass through, got %q", got)
	}

	// Empty stays empty.
	if got := svc.PresignStoredObjectKey(ctx, "", time.Minute); got != "" {
		t.Fatalf("empty must stay empty, got %q", got)
	}
}
