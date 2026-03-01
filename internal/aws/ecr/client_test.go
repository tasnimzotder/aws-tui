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

	tests := []struct {
		name           string
		mockRepos      [][]ecrtypes.Repository // pages of repos
		mockNextTokens []*string               // next token per page (nil = last)
		mockImageCount int
		wantRepoCount  int
		wantNames      []string
		wantImgCount   int
	}{
		{
			name: "single page single repo",
			mockRepos: [][]ecrtypes.Repository{
				{
					{
						RepositoryName: awssdk.String("my-app"),
						RepositoryUri:  awssdk.String("123456.dkr.ecr.us-east-1.amazonaws.com/my-app"),
						CreatedAt:      &created,
					},
				},
			},
			mockNextTokens: []*string{nil},
			mockImageCount: 2,
			wantRepoCount:  1,
			wantNames:      []string{"my-app"},
			wantImgCount:   2,
		},
		{
			name: "two pages of repos",
			mockRepos: [][]ecrtypes.Repository{
				{{RepositoryName: awssdk.String("repo-1"), RepositoryUri: awssdk.String("123.dkr.ecr.us-east-1.amazonaws.com/repo-1"), CreatedAt: &created}},
				{{RepositoryName: awssdk.String("repo-2"), RepositoryUri: awssdk.String("123.dkr.ecr.us-east-1.amazonaws.com/repo-2"), CreatedAt: &created}},
			},
			mockNextTokens: []*string{awssdk.String("page2"), nil},
			mockImageCount: 1,
			wantRepoCount:  2,
			wantNames:      []string{"repo-1", "repo-2"},
			wantImgCount:   1,
		},
		{
			name:           "empty repositories",
			mockRepos:      [][]ecrtypes.Repository{{}},
			mockNextTokens: []*string{nil},
			mockImageCount: 0,
			wantRepoCount:  0,
			wantNames:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callIdx := 0
			mock := &mockECRAPI{
				describeRepositoriesFunc: func(ctx context.Context, params *awsecr.DescribeRepositoriesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeRepositoriesOutput, error) {
					idx := callIdx
					callIdx++
					out := &awsecr.DescribeRepositoriesOutput{
						Repositories: tt.mockRepos[idx],
					}
					if tt.mockNextTokens[idx] != nil {
						out.NextToken = tt.mockNextTokens[idx]
					}
					return out, nil
				},
				describeImagesFunc: func(ctx context.Context, params *awsecr.DescribeImagesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeImagesOutput, error) {
					images := make([]ecrtypes.ImageDetail, tt.mockImageCount)
					for i := range images {
						images[i] = ecrtypes.ImageDetail{ImageTags: []string{"tag"}}
					}
					return &awsecr.DescribeImagesOutput{ImageDetails: images}, nil
				},
			}

			client := NewClient(mock)
			repos, err := client.ListRepositories(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(repos) != tt.wantRepoCount {
				t.Fatalf("repo count = %d, want %d", len(repos), tt.wantRepoCount)
			}
			for i, name := range tt.wantNames {
				if repos[i].Name != name {
					t.Errorf("repos[%d].Name = %s, want %s", i, repos[i].Name, name)
				}
			}
			if tt.wantRepoCount > 0 && repos[0].ImageCount != tt.wantImgCount {
				t.Errorf("ImageCount = %d, want %d", repos[0].ImageCount, tt.wantImgCount)
			}
		})
	}
}
