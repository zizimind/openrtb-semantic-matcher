package logging

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/segmentio/kafka-go"
)

// Logger interface for bid request/response logging
type Logger interface {
	LogRequest(requestID string, data []byte) error
	LogResponse(requestID string, data []byte) error
	Close() error
}

// KafkaConfig holds Kafka connection settings
type KafkaConfig struct {
	Brokers        []string
	TopicRequests  string
	TopicResponses string
	Compression    string
	OutputDir      string
}

// KafkaLogger logs to Kafka with zstd compression
type KafkaLogger struct {
	requestWriter  *kafka.Writer
	responseWriter *kafka.Writer
	encoder        *zstd.Encoder
	outputDir      string
	mu             sync.Mutex
}

// NewKafkaLogger creates a new Kafka logger with zstd compression
func NewKafkaLogger(ctx context.Context, cfg KafkaConfig) (*KafkaLogger, error) {
	// Create output directory for local fallback
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return nil, err
	}

	// Create zstd encoder
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return nil, err
	}

	// Create Kafka writers
	requestWriter := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.TopicRequests,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		Async:        true,
		Compression:  kafka.Zstd,
	}

	responseWriter := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.TopicResponses,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		Async:        true,
		Compression:  kafka.Zstd,
	}

	return &KafkaLogger{
		requestWriter:  requestWriter,
		responseWriter: responseWriter,
		encoder:        encoder,
		outputDir:      cfg.OutputDir,
	}, nil
}

// LogRequest logs a bid request to Kafka
func (l *KafkaLogger) LogRequest(requestID string, data []byte) error {
	compressed := l.compress(data)

	msg := kafka.Message{
		Key:   []byte(requestID),
		Value: compressed,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{Key: "type", Value: []byte("request")},
			{Key: "compression", Value: []byte("zstd")},
		},
	}

	return l.requestWriter.WriteMessages(context.Background(), msg)
}

// LogResponse logs a bid response to Kafka
func (l *KafkaLogger) LogResponse(requestID string, data []byte) error {
	compressed := l.compress(data)

	msg := kafka.Message{
		Key:   []byte(requestID),
		Value: compressed,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{Key: "type", Value: []byte("response")},
			{Key: "compression", Value: []byte("zstd")},
		},
	}

	return l.responseWriter.WriteMessages(context.Background(), msg)
}

// compress compresses data using zstd
func (l *KafkaLogger) compress(data []byte) []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.encoder.EncodeAll(data, nil)
}

// Close closes all Kafka writers
func (l *KafkaLogger) Close() error {
	if l.requestWriter != nil {
		l.requestWriter.Close()
	}
	if l.responseWriter != nil {
		l.responseWriter.Close()
	}
	if l.encoder != nil {
		l.encoder.Close()
	}
	return nil
}

// FileLogger is a fallback logger that writes to local files
type FileLogger struct {
	outputDir   string
	requestFile *os.File
	responseFile *os.File
	encoder     *zstd.Encoder
	mu          sync.Mutex
}

// NewFileLogger creates a file-based logger with zstd compression
func NewFileLogger(outputDir string) *FileLogger {
	os.MkdirAll(outputDir, 0755)

	encoder, _ := zstd.NewWriter(nil)

	// Create compressed output files
	reqFile, _ := os.OpenFile(
		filepath.Join(outputDir, "bid_requests.zst"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	respFile, _ := os.OpenFile(
		filepath.Join(outputDir, "bid_responses.zst"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)

	return &FileLogger{
		outputDir:    outputDir,
		requestFile:  reqFile,
		responseFile: respFile,
		encoder:      encoder,
	}
}

// LogRequest writes request to local file
func (l *FileLogger) LogRequest(requestID string, data []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.requestFile == nil {
		return nil
	}

	// Add newline delimiter
	data = append(data, '\n')
	compressed := l.encoder.EncodeAll(data, nil)
	_, err := l.requestFile.Write(compressed)
	return err
}

// LogResponse writes response to local file
func (l *FileLogger) LogResponse(requestID string, data []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.responseFile == nil {
		return nil
	}

	data = append(data, '\n')
	compressed := l.encoder.EncodeAll(data, nil)
	_, err := l.responseFile.Write(compressed)
	return err
}

// Close closes all file handles
func (l *FileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.requestFile != nil {
		l.requestFile.Close()
	}
	if l.responseFile != nil {
		l.responseFile.Close()
	}
	if l.encoder != nil {
		l.encoder.Close()
	}
	return nil
}
