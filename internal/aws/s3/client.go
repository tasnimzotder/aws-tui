package s3

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3API interface {
	ListBuckets(ctx context.Context, params *awss3.ListBucketsInput, optFns ...func(*awss3.Options)) (*awss3.ListBucketsOutput, error)
	GetBucketLocation(ctx context.Context, params *awss3.GetBucketLocationInput, optFns ...func(*awss3.Options)) (*awss3.GetBucketLocationOutput, error)
	ListObjectsV2(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error)
}

type Client struct {
	api S3API
}

func NewClient(api S3API) *Client {
	return &Client{api: api}
}

func (c *Client) ListBuckets(ctx context.Context) ([]S3Bucket, error) {
	out, err := c.api.ListBuckets(ctx, &awss3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("ListBuckets: %w", err)
	}

	buckets := make([]S3Bucket, len(out.Buckets))

	// Resolve regions concurrently, bounded to 10 goroutines
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, b := range out.Buckets {
		var createdAt time.Time
		if b.CreationDate != nil {
			createdAt = *b.CreationDate
		}
		buckets[i] = S3Bucket{
			Name:      aws.ToString(b.Name),
			CreatedAt: createdAt,
		}

		wg.Add(1)
		go func(idx int, name string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			locOut, err := c.api.GetBucketLocation(ctx, &awss3.GetBucketLocationInput{
				Bucket: aws.String(name),
			})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("GetBucketLocation(%s): %w", name, err)
				}
				mu.Unlock()
				return
			}

			region := string(locOut.LocationConstraint)
			if region == "" {
				region = "us-east-1"
			}

			mu.Lock()
			buckets[idx].Region = region
			mu.Unlock()
		}(i, aws.ToString(b.Name))
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return buckets, nil
}

func (c *Client) ListObjects(ctx context.Context, bucket, prefix, continuationToken, region string) (ListObjectsResult, error) {
	input := &awss3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(1000),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}
	if continuationToken != "" {
		input.ContinuationToken = aws.String(continuationToken)
	}

	var opts []func(*awss3.Options)
	if region != "" {
		opts = append(opts, func(o *awss3.Options) {
			o.Region = region
		})
	}

	out, err := c.api.ListObjectsV2(ctx, input, opts...)
	if err != nil {
		return ListObjectsResult{}, fmt.Errorf("ListObjectsV2: %w", err)
	}

	var objects []S3Object

	// Common prefixes first
	for _, cp := range out.CommonPrefixes {
		objects = append(objects, S3Object{
			Key:      aws.ToString(cp.Prefix),
			IsPrefix: true,
		})
	}

	// Then regular objects
	for _, obj := range out.Contents {
		var lastModified time.Time
		if obj.LastModified != nil {
			lastModified = *obj.LastModified
		}
		objects = append(objects, S3Object{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: lastModified,
			StorageClass: string(obj.StorageClass),
		})
	}

	result := ListObjectsResult{Objects: objects}
	if out.IsTruncated != nil && *out.IsTruncated {
		result.NextToken = aws.ToString(out.NextContinuationToken)
	}

	return result, nil
}
