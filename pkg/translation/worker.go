// Package translation translates chat and transcript text off the realtime path.
package translation

import (
	"context"
	"time"
)

type Kind string

const (
	ChatEvent       Kind = "chat"
	TranscriptEvent Kind = "transcript"
)

// Job is a text event that can be translated independently of its source.
type Job struct {
	EventID        string
	RoomID         string
	Kind           Kind
	Text           string
	SourceLanguage string
	TargetLanguage string
}

type Result struct {
	Job
	Text             string
	DetectedLanguage string
}

type Translator func(context.Context, Job) (Result, error)

type Worker struct {
	jobs      chan Job
	timeout   time.Duration
	translate Translator
	publish   func(Result)
}

// NewWorker makes a bounded, best-effort worker. A nil translator disables it.
func NewWorker(queueSize int, timeout time.Duration, translate Translator, publish func(Result)) *Worker {
	if queueSize < 1 {
		queueSize = 1
	}
	if timeout <= 0 {
		timeout = time.Second
	}
	return &Worker{jobs: make(chan Job, queueSize), timeout: timeout, translate: translate, publish: publish}
}

// Submit never waits: callers keep their original event path if translation is unavailable.
func (w *Worker) Submit(job Job) bool {
	if w.translate == nil || job.Text == "" || job.TargetLanguage == "" {
		return false
	}
	select {
	case w.jobs <- job:
		return true
	default:
		return false
	}
}

func (w *Worker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-w.jobs:
			jobCtx, cancel := context.WithTimeout(ctx, w.timeout)
			result, err := w.translate(jobCtx, job)
			timedOut := jobCtx.Err() != nil
			cancel()
			if err == nil && !timedOut && w.publish != nil {
				w.publish(result)
			}
		}
	}
}
