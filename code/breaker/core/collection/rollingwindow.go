package collection

import (
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/timex"
)

type (
	// RollingWindowOption let callers customize the RollingWindow.
	RollingWindowOption func(rollingWindow *RollingWindow)

	// RollingWindow defines a rolling window to calculate the events in buckets with time interval.
	RollingWindow struct {
		lock          sync.RWMutex
		size          int           // 窗口大小
		win           *window       // 窗口
		interval      time.Duration // 窗口间隔周期
		offset        int           // 窗口 offset
		ignoreCurrent bool
		lastTime      time.Duration // start time of the last bucket
	}
)

// 初始化滑动窗口
// NewRollingWindow returns a RollingWindow that with size buckets and time interval,
// use opts to customize the RollingWindow.
func NewRollingWindow(size int, interval time.Duration, opts ...RollingWindowOption) *RollingWindow {
	if size < 1 {
		panic("size must be greater than 0")
	}
	w := &RollingWindow{
		size:     size,
		win:      newWindow(size),
		interval: interval,
		lastTime: timex.Now(),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Add adds value to current bucket.
func (rw *RollingWindow) Add(v float64) {
	rw.lock.Lock()
	defer rw.lock.Unlock()
	// 滑动的动作发生在此
	rw.updateOffset()
	rw.win.add(rw.offset, v)
}

// 遍历所有 bucket
// Reduce runs fn on all buckets, ignore current bucket if ignoreCurrent was set.
func (rw *RollingWindow) Reduce(fn func(b *Bucket)) {
	rw.lock.RLock()
	defer rw.lock.RUnlock()

	var diff int
	span := rw.span()
	// ignore current bucket, because of partial data
	if span == 0 && rw.ignoreCurrent {
		diff = rw.size - 1
	} else {
		diff = rw.size - span
	}
	if diff > 0 {
		offset := (rw.offset + span + 1) % rw.size
		rw.win.reduce(offset, diff, fn)
	}
}

// 获取上一个更新时间跨越了几个 bucket
func (rw *RollingWindow) span() int {
	offset := int(timex.Since(rw.lastTime) / rw.interval)
	if 0 <= offset && offset < rw.size {
		return offset
	}

	return rw.size
}

// 窗口滑动
func (rw *RollingWindow) updateOffset() {
	span := rw.span()
	if span <= 0 {
		return
	}

	offset := rw.offset
	// reset expired buckets
	// 重置过期的 bucket
	for i := 0; i < span; i++ {
		rw.win.resetBucket((offset + i + 1) % rw.size)
	}

	rw.offset = (offset + span) % rw.size
	now := timex.Now()
	// align to interval time boundary
	// 更新时间(当前时间-上次时间余数)
	rw.lastTime = now - (now-rw.lastTime)%rw.interval
}

// Bucket defines the bucket that holds sum and num of additions.
// 桶
// sum 数据总量
// count 桶数量
type Bucket struct {
	Sum   float64
	Count int64
}

// 添加数据
func (b *Bucket) add(v float64) {
	b.Sum += v
	b.Count++
}

// 桶重置
func (b *Bucket) reset() {
	b.Sum = 0
	b.Count = 0
}

// 滑动窗口（环形数组）
type window struct {
	buckets []*Bucket
	size    int
}

// 初始化窗口
func newWindow(size int) *window {
	buckets := make([]*Bucket, size)
	for i := 0; i < size; i++ {
		buckets[i] = new(Bucket)
	}
	return &window{
		buckets: buckets,
		size:    size,
	}
}

// 往执行的 bucket 加入指定的指标
func (w *window) add(offset int, v float64) {
	w.buckets[offset%w.size].add(v)
}

// 遍历 buckets
func (w *window) reduce(start, count int, fn func(b *Bucket)) {
	for i := 0; i < count; i++ {
		fn(w.buckets[(start+i)%w.size])
	}
}

// 重置 buckets
func (w *window) resetBucket(offset int) {
	w.buckets[offset%w.size].reset()
}

// IgnoreCurrentBucket lets the Reduce call ignore current bucket.
// 让 Reduce 调用忽略当前bucket
func IgnoreCurrentBucket() RollingWindowOption {
	return func(w *RollingWindow) {
		w.ignoreCurrent = true
	}
}
