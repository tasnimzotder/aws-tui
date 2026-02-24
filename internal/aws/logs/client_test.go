package logs

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwltypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLogsAPI struct {
	getLogEventsFunc func(ctx context.Context, params *cloudwatchlogs.GetLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetLogEventsOutput, error)
}

func (m *mockLogsAPI) GetLogEvents(ctx context.Context, params *cloudwatchlogs.GetLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetLogEventsOutput, error) {
	return m.getLogEventsFunc(ctx, params, optFns...)
}

func TestGetLatestLogEvents(t *testing.T) {
	mock := &mockLogsAPI{
		getLogEventsFunc: func(ctx context.Context, params *cloudwatchlogs.GetLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetLogEventsOutput, error) {
			assert.Equal(t, "/ecs/my-app", *params.LogGroupName)
			assert.Equal(t, "ecs/web/abc123", *params.LogStreamName)
			assert.Equal(t, int32(50), *params.Limit)
			assert.Equal(t, false, *params.StartFromHead)

			return &cloudwatchlogs.GetLogEventsOutput{
				Events: []cwltypes.OutputLogEvent{
					{Timestamp: awssdk.Int64(1700000000000), Message: awssdk.String("Starting server...")},
					{Timestamp: awssdk.Int64(1700000001000), Message: awssdk.String("Listening on :8080")},
				},
				NextForwardToken: awssdk.String("fwd-token-1"),
			}, nil
		},
	}

	client := NewClient(mock)
	events, token, err := client.GetLatestLogEvents(context.Background(), "/ecs/my-app", "ecs/web/abc123", 50)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "Starting server...", events[0].Message)
	assert.Equal(t, "Listening on :8080", events[1].Message)
	assert.Equal(t, "fwd-token-1", token)
}

func TestGetLogEventsSince(t *testing.T) {
	mock := &mockLogsAPI{
		getLogEventsFunc: func(ctx context.Context, params *cloudwatchlogs.GetLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetLogEventsOutput, error) {
			assert.Equal(t, "fwd-token-1", *params.NextToken)
			assert.Equal(t, true, *params.StartFromHead)

			return &cloudwatchlogs.GetLogEventsOutput{
				Events: []cwltypes.OutputLogEvent{
					{Timestamp: awssdk.Int64(1700000002000), Message: awssdk.String("Request received")},
				},
				NextForwardToken: awssdk.String("fwd-token-2"),
			}, nil
		},
	}

	client := NewClient(mock)
	events, token, err := client.GetLogEventsSince(context.Background(), "/ecs/my-app", "ecs/web/abc123", "fwd-token-1")
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "Request received", events[0].Message)
	assert.Equal(t, "fwd-token-2", token)
}
