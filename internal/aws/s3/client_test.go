package s3

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type mockS3API struct {
	listBucketsFunc       func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error)
	getBucketLocationFunc func(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error)
	listObjectsV2Func    func(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error)
	getObjectFunc         func(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
}

func (m *mockS3API) ListBuckets(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error) {
	return m.listBucketsFunc(ctx, params, optFns...)
}

func (m *mockS3API) GetBucketLocation(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error) {
	return m.getBucketLocationFunc(ctx, params, optFns...)
}

func (m *mockS3API) ListObjectsV2(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error) {
	return m.listObjectsV2Func(ctx, params, optFns...)
}

func (m *mockS3API) GetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	return m.getObjectFunc(ctx, params, optFns...)
}

func TestListBuckets(t *testing.T) {
	created1 := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	created2 := time.Date(2025, 6, 20, 0, 0, 0, 0, time.UTC)
	created3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name              string
		listBucketsFunc   func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error)
		getBucketLocFunc  func(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error)
		wantErr           bool
		wantErrContains   []string
		wantBucketCount   int
		wantFirstName     string
		wantFirstRegion   string
		wantFirstCreated  time.Time
		wantSecondName    string
		wantSecondRegion  string
	}{
		{
			name: "success with region",
			listBucketsFunc: func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error) {
				return &awss3.ListBucketsOutput{
					Buckets: []s3types.Bucket{
						{Name: awssdk.String("bucket-a"), CreationDate: &created1},
						{Name: awssdk.String("bucket-b"), CreationDate: &created2},
					},
				}, nil
			},
			getBucketLocFunc: func(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error) {
				switch awssdk.ToString(params.Bucket) {
				case "bucket-a":
					return &awss3.GetBucketLocationOutput{
						LocationConstraint: s3types.BucketLocationConstraintEuWest1,
					}, nil
				default:
					return &awss3.GetBucketLocationOutput{
						LocationConstraint: "",
					}, nil
				}
			},
			wantBucketCount:  2,
			wantFirstName:    "bucket-a",
			wantFirstRegion:  "eu-west-1",
			wantFirstCreated: created1,
			wantSecondName:   "bucket-b",
			wantSecondRegion: "us-east-1",
		},
		{
			name: "empty location defaults to us-east-1",
			listBucketsFunc: func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error) {
				return &awss3.ListBucketsOutput{
					Buckets: []s3types.Bucket{
						{Name: awssdk.String("us-bucket"), CreationDate: &created3},
					},
				}, nil
			},
			getBucketLocFunc: func(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error) {
				return &awss3.GetBucketLocationOutput{
					LocationConstraint: "",
				}, nil
			},
			wantBucketCount: 1,
			wantFirstName:   "us-bucket",
			wantFirstRegion: "us-east-1",
		},
		{
			name: "API error",
			listBucketsFunc: func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error) {
				return nil, fmt.Errorf("access denied")
			},
			wantErr:         true,
			wantErrContains: []string{"ListBuckets"},
		},
		{
			name: "location error",
			listBucketsFunc: func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error) {
				return &awss3.ListBucketsOutput{
					Buckets: []s3types.Bucket{
						{Name: awssdk.String("fail-bucket"), CreationDate: &created3},
					},
				}, nil
			},
			getBucketLocFunc: func(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error) {
				return nil, fmt.Errorf("forbidden")
			},
			wantErr:         true,
			wantErrContains: []string{"GetBucketLocation", "fail-bucket"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockS3API{
				listBucketsFunc:       tt.listBucketsFunc,
				getBucketLocationFunc: tt.getBucketLocFunc,
			}
			client := NewClient(mock)
			buckets, err := client.ListBuckets(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				for _, substr := range tt.wantErrContains {
					if !strings.Contains(err.Error(), substr) {
						t.Errorf("error should contain %q, got: %v", substr, err)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(buckets) != tt.wantBucketCount {
				t.Fatalf("expected %d buckets, got %d", tt.wantBucketCount, len(buckets))
			}
			if tt.wantBucketCount > 0 {
				if buckets[0].Name != tt.wantFirstName {
					t.Errorf("Name = %s, want %s", buckets[0].Name, tt.wantFirstName)
				}
				if buckets[0].Region != tt.wantFirstRegion {
					t.Errorf("Region = %s, want %s", buckets[0].Region, tt.wantFirstRegion)
				}
				if !tt.wantFirstCreated.IsZero() && !buckets[0].CreatedAt.Equal(tt.wantFirstCreated) {
					t.Errorf("CreatedAt = %v, want %v", buckets[0].CreatedAt, tt.wantFirstCreated)
				}
			}
			if tt.wantBucketCount > 1 {
				if buckets[1].Name != tt.wantSecondName {
					t.Errorf("Name = %s, want %s", buckets[1].Name, tt.wantSecondName)
				}
				if buckets[1].Region != tt.wantSecondRegion {
					t.Errorf("Region = %s, want %s", buckets[1].Region, tt.wantSecondRegion)
				}
			}
		})
	}
}

func TestListObjects_BasicListing(t *testing.T) {
	lastMod := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)

	mock := &mockS3API{
		listObjectsV2Func: func(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error) {
			return &awss3.ListObjectsV2Output{
				CommonPrefixes: []s3types.CommonPrefix{
					{Prefix: awssdk.String("photos/")},
				},
				Contents: []s3types.Object{
					{
						Key:          awssdk.String("readme.txt"),
						Size:         awssdk.Int64(1024),
						LastModified: &lastMod,
						StorageClass: s3types.ObjectStorageClassStandard,
					},
				},
				IsTruncated: awssdk.Bool(false),
			}, nil
		},
	}

	client := NewClient(mock)
	result, err := client.ListObjects(context.Background(), "my-bucket", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(result.Objects))
	}

	// First should be the prefix
	if result.Objects[0].Key != "photos/" {
		t.Errorf("Objects[0].Key = %s, want photos/", result.Objects[0].Key)
	}
	if !result.Objects[0].IsPrefix {
		t.Errorf("Objects[0].IsPrefix = false, want true")
	}

	// Second should be the regular object
	if result.Objects[1].Key != "readme.txt" {
		t.Errorf("Objects[1].Key = %s, want readme.txt", result.Objects[1].Key)
	}
	if result.Objects[1].Size != 1024 {
		t.Errorf("Objects[1].Size = %d, want 1024", result.Objects[1].Size)
	}
	if result.Objects[1].StorageClass != "STANDARD" {
		t.Errorf("Objects[1].StorageClass = %s, want STANDARD", result.Objects[1].StorageClass)
	}
	if result.Objects[1].IsPrefix {
		t.Errorf("Objects[1].IsPrefix = true, want false")
	}
	if result.NextToken != "" {
		t.Errorf("NextToken = %s, want empty", result.NextToken)
	}
}

func TestListObjects_Error(t *testing.T) {
	mock := &mockS3API{
		listObjectsV2Func: func(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error) {
			return nil, fmt.Errorf("no such bucket")
		},
	}

	client := NewClient(mock)
	_, err := client.ListObjects(context.Background(), "missing-bucket", "", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ListObjectsV2") {
		t.Errorf("error should wrap with ListObjectsV2 context, got: %v", err)
	}
}

func TestListObjects_Pagination(t *testing.T) {
	mock := &mockS3API{
		listObjectsV2Func: func(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error) {
			return &awss3.ListObjectsV2Output{
				Contents: []s3types.Object{
					{Key: awssdk.String("file1.txt"), Size: awssdk.Int64(100)},
				},
				IsTruncated:           awssdk.Bool(true),
				NextContinuationToken: awssdk.String("token-abc"),
			}, nil
		},
	}

	client := NewClient(mock)
	result, err := client.ListObjects(context.Background(), "my-bucket", "prefix/", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NextToken != "token-abc" {
		t.Errorf("NextToken = %s, want token-abc", result.NextToken)
	}
	if len(result.Objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(result.Objects))
	}
}

func TestGetObject(t *testing.T) {
	mock := &mockS3API{
		getObjectFunc: func(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
			if awssdk.ToString(params.Bucket) != "my-bucket" {
				t.Errorf("Bucket = %s, want my-bucket", awssdk.ToString(params.Bucket))
			}
			if awssdk.ToString(params.Key) != "hello.txt" {
				t.Errorf("Key = %s, want hello.txt", awssdk.ToString(params.Key))
			}
			return &awss3.GetObjectOutput{
				Body: io.NopCloser(strings.NewReader("hello world")),
			}, nil
		},
	}

	client := NewClient(mock)
	data, err := client.GetObject(context.Background(), "my-bucket", "hello.txt", "us-east-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("data = %q, want %q", string(data), "hello world")
	}
}

func TestGetObject_Error(t *testing.T) {
	mock := &mockS3API{
		getObjectFunc: func(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
			return nil, fmt.Errorf("access denied")
		},
	}

	client := NewClient(mock)
	_, err := client.GetObject(context.Background(), "my-bucket", "secret.txt", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "GetObject") {
		t.Errorf("error should wrap with GetObject context, got: %v", err)
	}
}

func TestGetObjectStream(t *testing.T) {
	body := io.NopCloser(strings.NewReader("streaming content"))
	mock := &mockS3API{
		getObjectFunc: func(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
			return &awss3.GetObjectOutput{
				Body:          body,
				ContentLength: awssdk.Int64(17),
			}, nil
		},
	}
	client := NewClient(mock)
	reader, size, err := client.GetObjectStream(context.Background(), "bucket", "key.txt", "us-east-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()

	if size != 17 {
		t.Errorf("size = %d, want 17", size)
	}
	data, _ := io.ReadAll(reader)
	if string(data) != "streaming content" {
		t.Errorf("data = %q, want 'streaming content'", data)
	}
}

func TestGetObjectStream_Error(t *testing.T) {
	mock := &mockS3API{
		getObjectFunc: func(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
			return nil, fmt.Errorf("access denied")
		},
	}
	client := NewClient(mock)
	_, _, err := client.GetObjectStream(context.Background(), "bucket", "key.txt", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
