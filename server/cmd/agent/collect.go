package main

// collect.go Linux 轻量指标采集(/proc 只读,零依赖):CPU 占用差值/内存/根分区磁盘/
// 网络速率差值/负载/uptime。心跳周期采集一帧随心跳上报——服务端落 Redis 快照
// (资产查看抽屉数据源;完整历史管道为平台 slice3,与此无耦合)。

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// metricsFrame 心跳上报的一帧快照(字段与服务端 AgentMetrics 对齐)。
type metricsFrame struct {
	CpuPercent     float64 `json:"cpuPercent"`
	Loadavg1       float64 `json:"loadavg1"`
	MemTotalKB     uint64  `json:"memTotalKb"`
	MemAvailableKB uint64  `json:"memAvailableKb"`
	DiskTotalBytes uint64  `json:"diskTotalBytes"`
	DiskUsedBytes  uint64  `json:"diskUsedBytes"`
	NetInBps       float64 `json:"netInBps"`
	NetOutBps      float64 `json:"netOutBps"`
	UptimeSec      uint64  `json:"uptimeSec"`
}

// collector 差值状态(CPU/网络速率需要上一帧采样)。
type collector struct {
	mu       sync.Mutex
	lastStat *cpuStat  // 上次 CPU jiffies
	lastNet  *netStat  // 上次网络累计字节
	lastAt   time.Time // 上次采样时间
}

type cpuStat struct{ idle, total uint64 }
type netStat struct{ in, out uint64 }

var collectorInst = &collector{}

// collect 采集一帧(调用频率=心跳周期;首帧速率/CPU 无法差值返回 0)。
func (c *collector) collect() metricsFrame {
	c.mu.Lock()
	defer c.mu.Unlock()
	frame := metricsFrame{
		Loadavg1:  readLoadavg1(),
		UptimeSec: readUptime(),
	}
	readMeminfo(&frame.MemTotalKB, &frame.MemAvailableKB)
	frame.DiskTotalBytes, frame.DiskUsedBytes = readRootDisk()

	now := time.Now()
	if stat := readCpuStat(); stat != nil {
		if c.lastStat != nil && now.After(c.lastAt) {
			dTotal := stat.total - c.lastStat.total
			dIdle := stat.idle - c.lastStat.idle
			if dTotal > 0 {
				frame.CpuPercent = float64(dTotal-dIdle) * 100 / float64(dTotal)
			}
		}
		c.lastStat = stat
	}
	if ns := readNetStat(); ns != nil {
		if c.lastNet != nil && now.After(c.lastAt) {
			sec := now.Sub(c.lastAt).Seconds()
			if sec > 0 {
				frame.NetInBps = float64(ns.in-c.lastNet.in) / sec
				frame.NetOutBps = float64(ns.out-c.lastNet.out) / sec
			}
		}
		c.lastNet = ns
	}
	c.lastAt = now
	return frame
}

// readCpuStat /proc/stat 首行汇总(全部核)。
func readCpuStat() *cpuStat {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)[1:]
		var total, idle uint64
		for i, fs := range fields {
			v, _ := strconv.ParseUint(fs, 10, 64)
			total += v
			if i == 3 || i == 4 { // idle + iowait 视作非占用
				idle += v
			}
		}
		return &cpuStat{idle: idle, total: total}
	}
	return nil
}

// readMeminfo /proc/meminfo 的 MemTotal/MemAvailable。
func readMeminfo(totalKB, availKB *uint64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			*totalKB = parseMeminfoKB(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			*availKB = parseMeminfoKB(line)
		}
	}
}

func parseMeminfoKB(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, _ := strconv.ParseUint(fields[1], 10, 64)
	return v
}

// readRootDisk 根分区总量/已用(statfs;已用 = total-free_blocks×bs)。
func readRootDisk() (total, used uint64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return 0, 0
	}
	bs := uint64(st.Bsize)
	total = st.Blocks * bs
	free := st.Bfree * uint64(st.Bsize)
	// 保留块(root 保留 5%)不计入用户可用,但算"已用"更直观——用 Blocks-Bfree 口径
	used = total - free
	return total, used
}

// readNetStat /proc/net/dev 全部物理接口(排除 lo)收发累计字节。
func readNetStat() *netStat {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil
	}
	defer f.Close()
	ns := &netStat{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(parts[1]))
		if len(fields) < 9 {
			continue
		}
		in, _ := strconv.ParseUint(fields[0], 10, 64)
		out, _ := strconv.ParseUint(fields[8], 10, 64)
		ns.in += in
		ns.out += out
	}
	return ns
}

// readLoadavg1 /proc/loadavg 第一列。
func readLoadavg1() float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(fields[0], 64)
	return v
}

// readUptime /proc/uptime 第一列(秒)。
func readUptime() uint64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(fields[0], 64)
	return uint64(v)
}
