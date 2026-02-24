package logs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

// CloudWatchLogsAPI defines the subset of CloudWatch Logs API we use.
type CloudWatchLogsAPI interface {
	GetLogEvents(ctx context.Context, params *cloudwatchlogs.GetLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetLogEventsOutput, error)
}

// Client wraps the CloudWatch Logs API.
type Client struct {
	api CloudWatchLogsAPI
}

// NewClient creates a new logs client.
func NewClient(api CloudWatchLogsAPI) *Client {
	return &Client{api: api}
}

// GetLatestLogEvents retrieves the most recent log events from a stream.
func (c *Client) GetLatestLogEvents(ctx context.Context, logGroup, logStream string, limit int) ([]LogEvent, string, error) {
	out, err := c.api.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(logStream),
		Limit:         aws.Int32(int32(limit)),
		StartFromHead: aws.Bool(false),
	})
	if err != nil {
		return nil, "", fmt.Errorf("GetLogEvents: %w", err)
	}

	events := make([]LogEvent, len(out.Events))
	for i, e := range out.Events {
		events[i] = LogEvent{
			Timestamp: time.UnixMilli(aws.ToInt64(e.Timestamp)),
			Message:   aws.ToString(e.Message),
		}
	}

	var token string
	if out.NextForwardToken != nil {
		token = *out.NextForwardToken
	}

	return events, token, nil
}

// GetLogEventsSince retrieves new log events using a forward token from a previous call.
func (c *Client) GetLogEventsSince(ctx context.Context, logGroup, logStream, forwardToken string) ([]LogEvent, string, error) {
	out, err := c.api.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(logStream),
		NextToken:     aws.String(forwardToken),
		StartFromHead: aws.Bool(true),
	})
	if err != nil {
		return nil, "", fmt.Errorf("GetLogEvents: %w", err)
	}

	events := make([]LogEvent, len(out.Events))
	for i, e := range out.Events {
		events[i] = LogEvent{
			Timestamp: time.UnixMilli(aws.ToInt64(e.Timestamp)),
			Message:   aws.ToString(e.Message),
		}
	}

	var token string
	if out.NextForwardToken != nil {
		token = *out.NextForwardToken
	}

	return events, token, nil
}
