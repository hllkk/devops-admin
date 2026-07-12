// Package snowflake 提供自实现的雪花算法（Snowflake）ID 生成器。
//
// 64-bit ID 位分配（经典 Twitter 方案）：
//
//	1 bit  符号位（恒 0）
//	41 bit 毫秒时间戳（相对 epoch，约 69 年）
//	10 bit worker id（最多 1024 个节点）
//	12 bit 序列号（每毫秒最多 4096）
//
// 通过 MustInit 在启动期初始化全局节点，业务侧调用 NextID 生成主键。
package snowflake

import (
	"errors"
	"sync"
	"time"
)

const (
	workerBits  = 10
	sequenceBits = 12

	workerMax   = int64(-1) ^ (int64(-1) << workerBits)   // 1023
	sequenceMax = int64(-1) ^ (int64(-1) << sequenceBits) // 4095

	timeShift   = workerBits + sequenceBits // 22
	workerShift = sequenceBits              // 12

	// clockBackwardMax 时钟回拨容忍阈值：阈值内自旋等待追上，超过则报错。
	clockBackwardMax = 5 * time.Millisecond
)

var (
	// node 全局雪花节点单例，由 MustInit 初始化。
	node     *snowflakeNode
	nodeOnce sync.Once
)

var (
	// ErrClockMovedBackward 时钟回拨超过容忍阈值。
	ErrClockMovedBackward = errors.New("snowflake: clock moved backwards")
	// ErrNotInitialized 未调用 MustInit 就生成 ID。
	ErrNotInitialized = errors.New("snowflake: node not initialized, call MustInit first")
)

// snowflakeNode 雪花节点。
type snowflakeNode struct {
	mu            sync.Mutex
	lastTimestamp int64 // 上次生成 ID 的时间戳（相对 epoch 的毫秒）
	workerID      int64
	sequence      int64
	epoch         time.Time
}

// MustInit 启动期初始化全局雪花节点。workerID 越界或 epoch 为零值时 panic
// （对齐 initialize.OtherInit 的 panic-on-error 风格）。重复调用幂等，以首次为准。
func MustInit(workerID int64, epoch time.Time) {
	if workerID < 0 || workerID > workerMax {
		panic("snowflake: worker id out of range [0, 1023]")
	}
	if epoch.IsZero() {
		panic("snowflake: epoch is zero")
	}
	nodeOnce.Do(func() {
		node = &snowflakeNode{
			workerID: workerID,
			epoch:    epoch,
		}
	})
}

// NextID 生成一个雪花 ID，并发安全。
func NextID() (int64, error) {
	if node == nil {
		return 0, ErrNotInitialized
	}
	return node.generate()
}

// MustNextID 生成 ID，出错 panic。便于在 GORM callback 等不便处理 error 的场景使用。
func MustNextID() int64 {
	id, err := NextID()
	if err != nil {
		panic(err)
	}
	return id
}

// newSnowflakeNode 构造一个独立节点（主要供测试使用，绕过全局单例）。
func newSnowflakeNode(workerID int64, epoch time.Time) *snowflakeNode {
	return &snowflakeNode{
		workerID: workerID,
		epoch:    epoch,
	}
}

// generate 生成一个雪花 ID。
func (n *snowflakeNode) generate() (int64, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := n.currentMillis()

	// 时钟回拨处理。
	if now < n.lastTimestamp {
		diff := time.Duration(n.lastTimestamp-now) * time.Millisecond
		if diff <= clockBackwardMax {
			// 阈值内：等待追上 lastTimestamp（time.Sleep 保证醒来后 now >= lastTimestamp）。
			time.Sleep(diff)
			now = n.currentMillis()
		} else {
			return 0, ErrClockMovedBackward
		}
	}

	switch {
	case now == n.lastTimestamp:
		// 同一毫秒内序列号递增。
		n.sequence = (n.sequence + 1) & sequenceMax
		if n.sequence == 0 {
			// 当前毫秒序列耗尽，等待下一毫秒。
			now = n.nextMillis(n.lastTimestamp)
		}
	default:
		// 新毫秒，序列归零。
		n.sequence = 0
	}

	n.lastTimestamp = now

	return (now << timeShift) | (n.workerID << workerShift) | n.sequence, nil
}

// currentMillis 返回当前相对 epoch 的毫秒数。
func (n *snowflakeNode) currentMillis() int64 {
	return time.Since(n.epoch).Milliseconds()
}

// nextMillis 自旋等待到 ts 的下一毫秒。
func (n *snowflakeNode) nextMillis(ts int64) int64 {
	now := n.currentMillis()
	for now <= ts {
		now = n.currentMillis()
	}
	return now
}
