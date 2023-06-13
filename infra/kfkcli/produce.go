package kfkcli

import (
	"context"
	"io"
	"sync/atomic"

	"github.com/segmentio/kafka-go"
)

type Producer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
	acquire()
}

type shadowProducer struct {
	id     string
	dialer Dialer
	write  *kafka.Writer
	count  atomic.Int64
	closed atomic.Bool
}

func (fp *shadowProducer) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	return fp.write.WriteMessages(ctx, msgs...)
}

func (fp *shadowProducer) Close() error {
	if !fp.closed.CompareAndSwap(false, true) {
		return io.ErrClosedPipe
	}
	if count := fp.count.Add(-1); count <= 0 {
		fp.dialer.release(fp.id)
	}
	return nil
}

func (fp *shadowProducer) acquire() {
	fp.count.Add(1)
}
