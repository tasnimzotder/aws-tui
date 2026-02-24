package ecr

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsecr "github.com/aws/aws-sdk-go-v2/service/ecr"
)

type ECRAPI interface {
	DescribeRepositories(ctx context.Context, params *awsecr.DescribeRepositoriesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeRepositoriesOutput, error)
	DescribeImages(ctx context.Context, params *awsecr.DescribeImagesInput, optFns ...func(*awsecr.Options)) (*awsecr.DescribeImagesOutput, error)
}

type Client struct {
	api ECRAPI
}

func NewClient(api ECRAPI) *Client {
	return &Client{api: api}
}

func (c *Client) ListRepositories(ctx context.Context) ([]ECRRepo, error) {
	var repos []ECRRepo
	var nextToken *string

	for {
		out, err := c.api.DescribeRepositories(ctx, &awsecr.DescribeRepositoriesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeRepositories: %w", err)
		}

		for _, r := range out.Repositories {
			var createdAt time.Time
			if r.CreatedAt != nil {
				createdAt = *r.CreatedAt
			}
			repos = append(repos, ECRRepo{
				Name:      aws.ToString(r.RepositoryName),
				URI:       aws.ToString(r.RepositoryUri),
				CreatedAt: createdAt,
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	// Get image counts per repo
	for i, repo := range repos {
		var count int
		var imgToken *string
		for {
			imgOut, err := c.api.DescribeImages(ctx, &awsecr.DescribeImagesInput{
				RepositoryName: aws.String(repo.Name),
				NextToken:      imgToken,
			})
			if err != nil {
				break
			}
			count += len(imgOut.ImageDetails)
			if imgOut.NextToken == nil {
				break
			}
			imgToken = imgOut.NextToken
		}
		repos[i].ImageCount = count
	}

	return repos, nil
}

func (c *Client) ListImages(ctx context.Context, repoName string) ([]ECRImage, error) {
	var images []ECRImage
	var nextToken *string

	for {
		out, err := c.api.DescribeImages(ctx, &awsecr.DescribeImagesInput{
			RepositoryName: aws.String(repoName),
			NextToken:      nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeImages: %w", err)
		}

		for _, img := range out.ImageDetails {
			digest := aws.ToString(img.ImageDigest)
			if parts := strings.SplitN(digest, ":", 2); len(parts) == 2 && len(parts[1]) > 12 {
				digest = parts[0] + ":" + parts[1][:12]
			}

			var pushedAt time.Time
			if img.ImagePushedAt != nil {
				pushedAt = *img.ImagePushedAt
			}

			var sizeMB float64
			if img.ImageSizeInBytes != nil {
				sizeMB = float64(*img.ImageSizeInBytes) / (1024 * 1024)
			}

			images = append(images, ECRImage{
				Tags:     img.ImageTags,
				Digest:   digest,
				SizeMB:   sizeMB,
				PushedAt: pushedAt,
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].PushedAt.After(images[j].PushedAt)
	})

	return images, nil
}
