package recording

import (
	"context"
	"sync"

	"github.com/psds-microservice/recording-service/pkg/gen/recording_service"
	"github.com/psds-microservice/session-manager-service/pkg/gen/session_manager_service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// StreamRecorder sends a copy of the client stream to recording-service and sets recording_url in session-manager.
type StreamRecorder interface {
	WriteChunk(ctx context.Context, sessionID string, data []byte)
	EndSession(ctx context.Context, sessionID string) // finalizes recording, gets URL, calls session-manager SetRecordingUrl
}

// Client implements StreamRecorder using gRPC to recording-service and session-manager.
type Client struct {
	recordingAddr string
	sessionAddr   string
	log           *zap.Logger
	mu            sync.Mutex
	streams       map[string]recording_service.RecordingService_IngestStreamClient
	recConn       *grpc.ClientConn
	sessConn      *grpc.ClientConn
}

// NewClient creates a recording client. Call Connect() before use, then Close() when done.
func NewClient(recordingAddr, sessionManagerAddr string, log *zap.Logger) *Client {
	return &Client{
		recordingAddr: recordingAddr,
		sessionAddr:   sessionManagerAddr,
		log:           log,
		streams:       make(map[string]recording_service.RecordingService_IngestStreamClient),
	}
}

// Connect establishes gRPC connections to recording-service and session-manager.
func (c *Client) Connect(ctx context.Context) error {
	recConn, err := grpc.NewClient(c.recordingAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	c.recConn = recConn
	sessConn, err := grpc.NewClient(c.sessionAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = recConn.Close()
		return err
	}
	c.sessConn = sessConn
	return nil
}

// Close closes gRPC connections and any open streams.
func (c *Client) Close() error {
	c.mu.Lock()
	for _, st := range c.streams {
		_ = st.CloseSend()
	}
	c.streams = make(map[string]recording_service.RecordingService_IngestStreamClient)
	c.mu.Unlock()
	if c.recConn != nil {
		_ = c.recConn.Close()
		c.recConn = nil
	}
	if c.sessConn != nil {
		_ = c.sessConn.Close()
		c.sessConn = nil
	}
	return nil
}

// WriteChunk sends a chunk to recording-service for the given session (opens stream on first chunk).
// Lock is held for the whole lookup/create and Send to avoid races with Close() and concurrent Send on the same stream.
func (c *Client) WriteChunk(ctx context.Context, sessionID string, data []byte) {
	if c.recConn == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	st, ok := c.streams[sessionID]
	if !ok {
		recClient := recording_service.NewRecordingServiceClient(c.recConn)
		var err error
		st, err = recClient.IngestStream(ctx)
		if err != nil {
			c.log.Warn("recording: start stream failed", zap.String("session_id", sessionID), zap.Error(err))
			return
		}
		c.streams[sessionID] = st
	}
	chunk := &recording_service.StreamChunk{SessionId: sessionID, Data: data, Last: false}
	if err := st.Send(chunk); err != nil {
		c.log.Warn("recording: send chunk failed", zap.String("session_id", sessionID), zap.Error(err))
		delete(c.streams, sessionID)
	}
}

// EndSession sends last chunk, closes stream, gets URL, and sets it in session-manager.
// Lock is held until stream is removed and final Send/CloseAndRecv are done so Close() cannot close connections meanwhile.
func (c *Client) EndSession(ctx context.Context, sessionID string) {
	if c.recConn == nil {
		return
	}
	c.mu.Lock()
	st, ok := c.streams[sessionID]
	if !ok {
		c.mu.Unlock()
		return
	}
	delete(c.streams, sessionID)
	_ = st.Send(&recording_service.StreamChunk{SessionId: sessionID, Last: true})
	res, err := st.CloseAndRecv()
	c.mu.Unlock()

	if err != nil {
		c.log.Warn("recording: close and recv failed", zap.String("session_id", sessionID), zap.Error(err))
		return
	}
	url := res.GetRecordingUrl()
	if url == "" && res.GetError() != "" {
		c.log.Warn("recording: error from service", zap.String("session_id", sessionID), zap.String("error", res.GetError()))
		return
	}
	if c.sessConn != nil && url != "" {
		smClient := session_manager_service.NewSessionManagerServiceClient(c.sessConn)
		_, err = smClient.SetRecordingUrl(ctx, &session_manager_service.SetRecordingUrlRequest{
			StreamSessionId: sessionID,
			RecordingUrl:    url,
		})
		if err != nil {
			c.log.Warn("session-manager: SetRecordingUrl failed", zap.String("session_id", sessionID), zap.Error(err))
		}
	}
}
