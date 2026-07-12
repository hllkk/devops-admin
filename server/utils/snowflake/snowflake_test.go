package snowflake

import (
	"sync"
	"testing"
	"time"
)

// testEpoch 固定测试起点：2024-01-01 00:00:00 UTC。
var testEpoch = time.Unix(1704067200, 0).UTC()

// TestGenerateConcurrentUnique 并发生成大量 ID，断言全部唯一。
func TestGenerateConcurrentUnique(t *testing.T) {
	n := newSnowflakeNode(1, testEpoch)

	const goroutines, perG = 100, 10000
	total := goroutines * perG

	var wg sync.WaitGroup
	ids := make(chan int64, total)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				id, err := n.generate()
				if err != nil {
					t.Errorf("generate error: %v", err)
					return
				}
				ids <- id
			}
		}()
	}
	wg.Wait()
	close(ids)

	seen := make(map[int64]struct{}, total)
	count := 0
	for id := range ids {
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate id: %d", id)
		}
		seen[id] = struct{}{}
		count++
	}
	if count != total {
		t.Fatalf("expected %d unique ids, got %d", total, count)
	}
}

// TestGenerateMonotonic 同一节点连续生成的 ID 单调递增。
func TestGenerateMonotonic(t *testing.T) {
	n := newSnowflakeNode(1, testEpoch)
	var prev int64
	for i := 0; i < 10000; i++ {
		id, err := n.generate()
		if err != nil {
			t.Fatalf("generate error: %v", err)
		}
		if id <= prev {
			t.Fatalf("id not monotonic at i=%d: prev=%d cur=%d", i, prev, id)
		}
		prev = id
	}
}

// TestWorkerIDEncoded 验证 worker id 被正确编码进 ID 的中 10 位。
func TestWorkerIDEncoded(t *testing.T) {
	const wid int64 = 7
	n := newSnowflakeNode(wid, testEpoch)
	id, err := n.generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	got := (id >> workerShift) & workerMax
	if got != wid {
		t.Fatalf("worker id not encoded: got %d want %d", got, wid)
	}
}

// TestGenerateTimestampEncoded 验证时间戳高位随时间推进。
func TestGenerateTimestampEncoded(t *testing.T) {
	n := newSnowflakeNode(0, testEpoch)
	id1, _ := n.generate()
	ts1 := id1 >> timeShift
	if ts1 <= 0 {
		t.Fatalf("timestamp not encoded: ts=%d", ts1)
	}
}

// TestMustInitPanicsOnInvalidWorker worker id 越界时 panic。
func TestMustInitPanicsOnInvalidWorker(t *testing.T) {
	for _, w := range []int64{-1, workerMax + 1} {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic for worker id %d", w)
				}
			}()
			MustInit(w, testEpoch)
		}()
	}
}

// TestMustInitPanicsOnZeroEpoch epoch 为零值时 panic。
func TestMustInitPanicsOnZeroEpoch(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for zero epoch")
			}
		}()
		MustInit(1, time.Time{})
	}()
}
