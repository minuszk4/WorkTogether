package translation

import (
	"context"
	"testing"
	"time"
)

func TestWorkerPublishesChatAndTranscriptTranslations(t *testing.T) {
	results := make(chan Result, 2)
	worker := NewWorker(2, time.Second, func(_ context.Context, job Job) (Result, error) {
		return Result{Job: job, Text: "xin chao"}, nil
	}, func(result Result) { results <- result })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Start(ctx)

	for _, kind := range []Kind{ChatEvent, TranscriptEvent} {
		if !worker.Submit(Job{EventID: string(kind), Kind: kind, Text: "hello", TargetLanguage: "vi"}) {
			t.Fatalf("Submit(%q) = false, want true", kind)
		}
	}

	for range 2 {
		select {
		case result := <-results:
			if result.Text != "xin chao" || result.TargetLanguage != "vi" {
				t.Fatalf("unexpected result: %#v", result)
			}
		case <-time.After(time.Second):
			t.Fatal("translation was not published")
		}
	}
}

func TestWorkerFailsOpenWhenDisabledFullOrTimedOut(t *testing.T) {
	if NewWorker(1, time.Second, nil, nil).Submit(Job{Text: "hello", TargetLanguage: "vi"}) {
		t.Fatal("disabled worker accepted a job")
	}

	worker := NewWorker(1, 10*time.Millisecond, func(ctx context.Context, job Job) (Result, error) {
		<-ctx.Done()
		return Result{Job: job, Text: "late"}, nil
	}, nil)
	if !worker.Submit(Job{Text: "hello", TargetLanguage: "vi"}) {
		t.Fatal("first job was rejected")
	}
	if worker.Submit(Job{Text: "again", TargetLanguage: "vi"}) {
		t.Fatal("full worker accepted a job")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Start(ctx)
	time.Sleep(50 * time.Millisecond)
}
