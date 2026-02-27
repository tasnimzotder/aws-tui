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

	mock := &mockS3API{
		listBucketsFunc: func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error) {
			return &awss3.ListBucketsOutput{
				Buckets: []s3types.Bucket{
					{Name: awssdk.String("bucket-a"), CreationDate: &created1},
					{Name: awssdk.String("bucket-b"), CreationDate: &created2},
				},
			}, nil
		},
		getBucketLocationFunc: func(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error) {
			switch awssdk.ToString(params.Bucket) {
			case "bucket-a":
				return &awss3.GetBucketLocationOutput{
					LocationConstraint: s3types.BucketLocationConstraintEuWest1,
				}, nil
			default:
				// Empty string means us-east-1
				return &awss3.GetBucketLocationOutput{
					LocationConstraint: "",
				}, nil
			}
		},
	}

	client := NewClient(mock)
	buckets, err := client.ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(buckets))
	}

	// bucket-a should be eu-west-1
	if buckets[0].Name != "bucket-a" {
		t.Errorf("Name = %s, want bucket-a", buckets[0].Name)
	}
	if buckets[0].Region != "eu-west-1" {
		t.Errorf("Region = %s, want eu-west-1", buckets[0].Region)
	}
	if !buckets[0].CreatedAt.Equal(created1) {
		t.Errorf("CreatedAt = %v, want %v", buckets[0].CreatedAt, created1)
	}

	// bucket-b should be us-east-1 (empty LocationConstraint)
	if buckets[1].Name != "bucket-b" {
		t.Errorf("Name = %s, want bucket-b", buckets[1].Name)
	}
	if buckets[1].Region != "us-east-1" {
		t.Errorf("Region = %s, want us-east-1", buckets[1].Region)
	}
}

func TestListBuckets_EmptyLocationIsUSEast1(t *testing.T) {
	created := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	mock := &mockS3API{
		listBucketsFunc: func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error) {
			return &awss3.ListBucketsOutput{
				Buckets: []s3types.Bucket{
					{Name: awssdk.String("us-bucket"), CreationDate: &created},
				},
			}, nil
		},
		getBucketLocationFunc: func(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error) {
			return &awss3.GetBucketLocationOutput{
				LocationConstraint: "",
			}, nil
		},
	}

	client := NewClient(mock)
	buckets, err := client.ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(buckets))
	}
	if buckets[0].Region != "us-east-1" {
		t.Errorf("Region = %s, want us-east-1", buckets[0].Region)
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

func TestListBuckets_APIError(t *testing.T) {
	mock := &mockS3API{
		listBucketsFunc: func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error) {
			return nil, fmt.Errorf("access denied")
		},
	}

	client := NewClient(mock)
	_, err := client.ListBuckets(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ListBuckets") {
		t.Errorf("error should wrap with ListBuckets context, got: %v", err)
	}
}

func TestListBuckets_LocationError(t *testing.T) {
	created := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	mock := &mockS3API{
		listBucketsFunc: func(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error) {
			return &awss3.ListBucketsOutput{
				Buckets: []s3types.Bucket{
					{Name: awssdk.String("fail-bucket"), CreationDate: &created},
				},
			}, nil
		},
		getBucketLocationFunc: func(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error) {
			return nil, fmt.Errorf("forbidden")
		},
	}

	client := NewClient(mock)
	_, err := client.ListBuckets(context.Background())
	if err == nil {
		t.Fatal("expected error from GetBucketLocation failure")
	}
	if !strings.Contains(err.Error(), "GetBucketLocation") {
		t.Errorf("error should contain GetBucketLocation context, got: %v", err)
	}
	if !strings.Contains(err.Error(), "fail-bucket") {
		t.Errorf("error should contain bucket name, got: %v", err)
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
