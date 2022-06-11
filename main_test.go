package main

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type batchWriter struct {
	flushCond *sync.Cond
	// typically set to 1MB
	flushBytes int64
	batchBytes int64
	buf        bytes.Buffer
}

func (w *batchWriter) WriteEvent(data []byte) error {
	w.flushCond.L.Lock()
	defer w.flushCond.L.Unlock()

	fmt.Println("batch bytes", w.batchBytes, "flush bytes", w.flushBytes)
	// typically this is 1MB
	if w.batchBytes >= w.flushBytes {
		// Given our concurrency model I'm sort of confused about how we could
		// possibly hit this condition.
		return fmt.Errorf("current batch size (%d) exceeds flush limit; cannot add more events", w.batchBytes)
	}
	w.buf.Write(data)
	w.batchBytes += int64(len(data))
	fmt.Println("signaling flush condition")
	w.flushCond.Signal()
	return nil
}

func (w *batchWriter) flushBatches() {
	fmt.Println("call flushBatches")
	// condition test function
	batchReady := func() bool {
		fmt.Println("checking if batch is ready")
		if w.batchBytes >= w.flushBytes {
			log.Printf("flushing because batch exceeds in-memory buffer limit (size=%d)", w.batchBytes)
			return true
		}
		return false
	}

	for {
		fmt.Println("attempting to lock flushCond")
		w.flushCond.L.Lock()
		for !batchReady() {
			fmt.Println("batch not ready, waiting on flushCond")
			w.flushCond.Wait()
		}
		fmt.Println("batch is ready")

		if err := w.flush(); err != nil {
			log.Printf("flush batch error: %v", err)
		}
		w.flushCond.L.Unlock()
	}
}

// w.flushCond.L must be held
func (w *batchWriter) flush() error {
	defer w.resetBatch()
	fmt.Println("flushing data", "len", w.buf.Len())
	return nil
}

// w.flushCond.L must be held
func (w *batchWriter) resetBatch() {
	w.batchBytes = 0
	w.buf.Reset()
}

func TestConcurrency(t *testing.T) {
	b := &batchWriter{
		flushCond:  sync.NewCond(new(sync.Mutex)),
		flushBytes: 50*20 - 1,
	}
	go b.flushBatches()
	// Ensure that flushBatches can sleep on the lock
	time.Sleep(2 * time.Second)
	var wg sync.WaitGroup
	var hit int32
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := bytes.Repeat([]byte{'a'}, 50)
			if err := b.WriteEvent(data); err != nil {
				fmt.Println("hit error")
				atomic.AddInt32(&hit, 1)
			}
		}()
	}
	wg.Wait()
	if hit > 0 {
		t.Fail()
	}
}
