package ecr

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsecr "github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

type mockECRAPI struct {
	describeRepositoriesFunc func(ctx context.Context, params *awsecr.DescribeRepositoriesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeRepositoriesOutput, error)
	describeImagesFunc       func(ctx context.Context, params *awsecr.DescribeImagesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeImagesOutput, error)
}

func (m *mockECRAPI) DescribeRepositories(ctx context.Context, params *awsecr.DescribeRepositoriesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeRepositoriesOutput, error) {
	return m.describeRepositoriesFunc(ctx, params, optFns...)
}
func (m *mockECRAPI) DescribeImages(ctx context.Context, params *awsecr.DescribeImagesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeImagesOutput, error) {
	return m.describeImagesFunc(ctx, params, optFns...)
}

func TestListRepositories(t *testing.T) {
	created := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	mock := &mockECRAPI{
		describeRepositoriesFunc: func(ctx context.Context, params *awsecr.DescribeRepositoriesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeRepositoriesOutput, error) {
			return &awsecr.DescribeRepositoriesOutput{
				Repositories: []ecrtypes.Repository{
					{
						RepositoryName: awssdk.String("my-app"),
						RepositoryUri:  awssdk.String("123456.dkr.ecr.us-east-1.amazonaws.com/my-app"),
						CreatedAt:      &created,
					},
				},
			}, nil
		},
		describeImagesFunc: func(ctx context.Context, params *awsecr.DescribeImagesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeImagesOutput, error) {
			return &awsecr.DescribeImagesOutput{
				ImageDetails: []ecrtypes.ImageDetail{
					{ImageTags: []string{"latest"}},
					{ImageTags: []string{"v1.0"}},
				},
			}, nil
		},
	}

	client := NewClient(mock)
	repos, err := client.ListRepositories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].Name != "my-app" {
		t.Errorf("Name = %s, want my-app", repos[0].Name)
	}
	if repos[0].ImageCount != 2 {
		t.Errorf("ImageCount = %d, want 2", repos[0].ImageCount)
	}
}

func TestListRepositories_Pagination(t *testing.T) {
	repoCalls := 0
	mock := &mockECRAPI{
		describeRepositoriesFunc: func(ctx context.Context, params *awsecr.DescribeRepositoriesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeRepositoriesOutput, error) {
			repoCalls++
			created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			if repoCalls == 1 {
				return &awsecr.DescribeRepositoriesOutput{
					Repositories: []ecrtypes.Repository{{
						RepositoryName: awssdk.String("repo-1"),
						RepositoryUri:  awssdk.String("123.dkr.ecr.us-east-1.amazonaws.com/repo-1"),
						CreatedAt:      &created,
					}},
					NextToken: awssdk.String("page2"),
				}, nil
			}
			return &awsecr.DescribeRepositoriesOutput{
				Repositories: []ecrtypes.Repository{{
					RepositoryName: awssdk.String("repo-2"),
					RepositoryUri:  awssdk.String("123.dkr.ecr.us-east-1.amazonaws.com/repo-2"),
					CreatedAt:      &created,
				}},
			}, nil
		},
		describeImagesFunc: func(ctx context.Context, params *awsecr.DescribeImagesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeImagesOutput, error) {
			return &awsecr.DescribeImagesOutput{
				ImageDetails: []ecrtypes.ImageDetail{{ImageTags: []string{"latest"}}},
			}, nil
		},
	}

	client := NewClient(mock)
	repos, err := client.ListRepositories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repoCalls != 2 {
		t.Errorf("expected 2 DescribeRepositories calls, got %d", repoCalls)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].Name != "repo-1" || repos[1].Name != "repo-2" {
		t.Errorf("unexpected repo names: %s, %s", repos[0].Name, repos[1].Name)
	}
}
