package s3

import "time"

type S3Bucket struct {
	Name      string
	Region    string
	CreatedAt time.Time
}

type S3Object struct {
	Key          string
	Size         int64
	LastModified time.Time
	StorageClass string
	IsPrefix     bool
}

type ListObjectsResult struct {
	Objects   []S3Object
	NextToken string
}
