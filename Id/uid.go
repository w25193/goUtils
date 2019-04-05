package idutils

// 自用类库
// +---------------+----------------+----------------+
// |timestamp(ms)42  | worker id(10) | sequence(12)	 |
// +---------------+----------------+----------------+
// id 生成算法库

import (
	"errors"
	"sync"
	"time"
)

const (
	//CEpoch 起始时间毫秒count
	CEpoch = 1474802888000
	//CWorkerIdBits 机器序列号 数据中心id+机器Id
	CWorkerIdBits = 10 // Num of WorkerId Bits
	//CSenquenceBits 每毫秒序列号
	CSenquenceBits = 12 // Num of Sequence Bits

	//CWorkerIdShift 偏移位数
	CWorkerIdShift = 12
	//CTimeStampShift 偏移位数
	CTimeStampShift = 22

	//CSequenceMask 每毫秒最大序列号
	CSequenceMask = 0xfff // equal as getSequenceMask()
	//CMaxWorker 最大机器序列号
	CMaxWorker = 0x3ff // equal as getMaxWorkerId()
)

// IdWorker Struct
type IdWorker struct {
	workerId      int64
	lastTimeStamp int64
	sequence      int64
	maxWorkerId   int64
	lock          *sync.Mutex
}

// 定义一个包级别的private实例变量
var worker *IdWorker

// 同步Once,保证每次调用时，只有第一次生效
var once sync.Once

//singleton
// NewId Func: Generate Given id
func NewId(workerid int64) (result int64) {
	once.Do(func() {
		worker, _ = NewIdWorker(10)
	})

	//产生新ID
	ResId, _ := worker.NextId()
	return ResId
}

// NewIdWorker Func: Generate NewIdWorker with Given workerid
func NewIdWorker(workerid int64) (iw *IdWorker, err error) {
	iw = new(IdWorker)

	iw.maxWorkerId = getMaxWorkerId()

	if workerid > iw.maxWorkerId || workerid < 0 {
		return nil, errors.New("worker not fit")
	}
	iw.workerId = workerid
	iw.lastTimeStamp = -1
	iw.sequence = 0
	iw.lock = new(sync.Mutex)
	return iw, nil
}

func getMaxWorkerId() int64 {
	return -1 ^ -1<<CWorkerIdBits
}

func getSequenceMask() int64 {
	return -1 ^ -1<<CSenquenceBits
}

// return in ms
func (iw *IdWorker) timeGen() int64 {
	return time.Now().UnixNano() / 1000 / 1000
}

func (iw *IdWorker) timeReGen(last int64) int64 {
	ts := time.Now().UnixNano() / 1000 / 1000
	for {
		if ts <= last {
			ts = iw.timeGen()
		} else {
			break
		}
	}
	return ts
}

// NewId Func: Generate next id
func (iw *IdWorker) NextId() (ts int64, err error) {
	iw.lock.Lock()
	defer iw.lock.Unlock()
	ts = iw.timeGen()
	if ts == iw.lastTimeStamp {
		iw.sequence = (iw.sequence + 1) & CSequenceMask
		if iw.sequence == 0 {
			ts = iw.timeReGen(ts)
		}
	} else {
		iw.sequence = 0
	}

	if ts < iw.lastTimeStamp {
		err = errors.New("Clock moved backwards, Refuse gen id")
		return 0, err
	}
	iw.lastTimeStamp = ts
	ts = (ts-CEpoch)<<CTimeStampShift | iw.workerId<<CWorkerIdShift | iw.sequence
	return ts, nil
}

// ParseId Func: reverse uid to timestamp, workid, seq
func ParseId(id int64) (t time.Time, ts int64, workerId int64, seq int64) {
	seq = id & CSequenceMask
	workerId = (id >> CWorkerIdShift) & CMaxWorker
	ts = (id >> CTimeStampShift) + CEpoch
	t = time.Unix(ts/1000, (ts%1000)*1000000)
	return
}
