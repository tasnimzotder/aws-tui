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
	tests := []struct {
		name           string
		logGroup       string
		logStream      string
		limit          int
		mockEvents     []cwltypes.OutputLogEvent
		mockToken      *string
		wantCount      int
		wantFirstMsg   string
		wantToken      string
	}{
		{
			name:      "returns multiple events",
			logGroup:  "/ecs/my-app",
			logStream: "ecs/web/abc123",
			limit:     50,
			mockEvents: []cwltypes.OutputLogEvent{
				{Timestamp: awssdk.Int64(1700000000000), Message: awssdk.String("Starting server...")},
				{Timestamp: awssdk.Int64(1700000001000), Message: awssdk.String("Listening on :8080")},
			},
			mockToken:    awssdk.String("fwd-token-1"),
			wantCount:    2,
			wantFirstMsg: "Starting server...",
			wantToken:    "fwd-token-1",
		},
		{
			name:       "returns empty events",
			logGroup:   "/ecs/my-app",
			logStream:  "ecs/web/empty",
			limit:      10,
			mockEvents: []cwltypes.OutputLogEvent{},
			mockToken:  awssdk.String(""),
			wantCount:  0,
			wantToken:  "",
		},
		{
			name:      "single event",
			logGroup:  "/ecs/svc",
			logStream: "ecs/task/xyz",
			limit:     1,
			mockEvents: []cwltypes.OutputLogEvent{
				{Timestamp: awssdk.Int64(1700000000000), Message: awssdk.String("Only line")},
			},
			mockToken:    awssdk.String("tok-single"),
			wantCount:    1,
			wantFirstMsg: "Only line",
			wantToken:    "tok-single",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLogsAPI{
				getLogEventsFunc: func(ctx context.Context, params *cloudwatchlogs.GetLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetLogEventsOutput, error) {
					assert.Equal(t, tt.logGroup, *params.LogGroupName)
					assert.Equal(t, tt.logStream, *params.LogStreamName)
					assert.Equal(t, int32(tt.limit), *params.Limit)
					assert.Equal(t, false, *params.StartFromHead)
					return &cloudwatchlogs.GetLogEventsOutput{
						Events:           tt.mockEvents,
						NextForwardToken: tt.mockToken,
					}, nil
				},
			}
			client := NewClient(mock)
			events, token, err := client.GetLatestLogEvents(context.Background(), tt.logGroup, tt.logStream, tt.limit)
			require.NoError(t, err)
			assert.Len(t, events, tt.wantCount)
			if tt.wantCount > 0 {
				assert.Equal(t, tt.wantFirstMsg, events[0].Message)
			}
			assert.Equal(t, tt.wantToken, token)
		})
	}
}

func TestGetLogEventsSince(t *testing.T) {
	tests := []struct {
		name         string
		token        string
		mockEvents   []cwltypes.OutputLogEvent
		mockToken    string
		wantCount    int
		wantFirstMsg string
		wantToken    string
	}{
		{
			name:  "returns new events",
			token: "fwd-token-1",
			mockEvents: []cwltypes.OutputLogEvent{
				{Timestamp: awssdk.Int64(1700000002000), Message: awssdk.String("Request received")},
			},
			mockToken:    "fwd-token-2",
			wantCount:    1,
			wantFirstMsg: "Request received",
			wantToken:    "fwd-token-2",
		},
		{
			name:       "no new events",
			token:      "fwd-token-2",
			mockEvents: []cwltypes.OutputLogEvent{},
			mockToken:  "fwd-token-2",
			wantCount:  0,
			wantToken:  "fwd-token-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLogsAPI{
				getLogEventsFunc: func(ctx context.Context, params *cloudwatchlogs.GetLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetLogEventsOutput, error) {
					assert.Equal(t, tt.token, *params.NextToken)
					assert.Equal(t, true, *params.StartFromHead)
					return &cloudwatchlogs.GetLogEventsOutput{
						Events:           tt.mockEvents,
						NextForwardToken: awssdk.String(tt.mockToken),
					}, nil
				},
			}
			client := NewClient(mock)
			events, token, err := client.GetLogEventsSince(context.Background(), "/ecs/my-app", "ecs/web/abc123", tt.token)
			require.NoError(t, err)
			assert.Len(t, events, tt.wantCount)
			if tt.wantCount > 0 {
				assert.Equal(t, tt.wantFirstMsg, events[0].Message)
			}
			assert.Equal(t, tt.wantToken, token)
		})
	}
}
