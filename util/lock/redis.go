package lock

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"

	"redisgo_task/config"
)

const (
	EXPIRE_TIME  = 5 * time.Second
	CRON_TIME    = 1 * time.Second
	RETRIES_TIME = 10 * time.Millisecond
)

type RedisLock struct {
	Host           string
	Expire         time.Duration
	Key            string
	Value          string
	Conn           redis.Conn
	Cron           time.Duration
	DoneExpireChan chan struct{}
	Retry          *Retry
}

func NewRedisLock(cfg *config.RedisInfo) (Locker, error) {
	cfg, isValid := Valid(cfg)
	if !isValid {
		fmt.Printf("RedisLock redisInfo param is error:%v \n", cfg)
		return nil, fmt.Errorf("param is valid,host or key is null")
	}
	conn, err := redis.Dial("tcp", cfg.Host)
	if err != nil {
		fmt.Printf("RedisLock redis dial is fail :%v \n", err)
		return nil, fmt.Errorf("RedisLock redis dial is fail ")
	}
	retry := NewRetry(0, cfg.RetriesCount, cfg.MonitorTryAll)
	return &RedisLock{
		Host:   cfg.Host,
		Key:    cfg.Key,
		Value:  cfg.Value,
		Expire: cfg.Expire.Duration,
		Cron:  cfg.Cron.Duration,
		Conn:  conn,
		Retry: retry,
	}, nil
}

func Valid(cfg *config.RedisInfo) (*config.RedisInfo, bool) {
	if cfg.Host == "" || cfg.Key == "" {
		fmt.Printf("NewRedisLock param is valid,host or key is null\n")
		return nil, false
	}
	if cfg.Value == "" {
		cfg.Value = "8292884c-a7a7-0050-9778-e47362a8f578"
	}
	if cfg.Expire.Duration == 0 {
		cfg.Expire.Duration = EXPIRE_TIME
	}
	if cfg.Cron.Duration == 0 {
		cfg.Cron.Duration = CRON_TIME
	}
	cfg.Value = cfg.Value + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	return cfg, true
}

func (lock *RedisLock) Lock() {
Wait:
	fmt.Printf("RedisLock redis Lock was Locking \n")
	_, err := redis.String(lock.Conn.Do("SET", lock.Key, lock.Value, "EX", int(lock.Expire), "NX"))
	if err == redis.ErrNil {
		// The lock was not successful, it already exists.
		fmt.Printf("RedisLock redis Lock was not successful, it already exists and now trying\n")
		sleepTime := lock.Retry.RetriesTime()
		if lock.Retry.RetriesTime() >= 0 {
			fmt.Printf("RedisLock redis Lock was trying time is:%v", sleepTime)
			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		} else {
			fmt.Printf("RedisLock redis Lock was not successful,but try index is used over")
			return
		}
	}
	if err != nil {
		fmt.Printf("RedisLock redis Lock was fail  :%v \n", err)
		goto Wait
	}
	fmt.Printf("RedisLock_redis_lock success : %v\n", err)
	lock.DoneExpireChan = make(chan struct{})
	go lock.watch(lock.DoneExpireChan)
	return
}

func (lock *RedisLock) watch(doneCh <-chan struct{}) {
	ticker := time.NewTicker(lock.Cron)
	for {
		select {
		case <-ticker.C:
			lock.ReExpire()
		case <-doneCh:
			return
		}
	}
}
func (lock *RedisLock) Unlock() {
	//保证原子性
	var updateRecordExpireScript = redis.NewScript(1, `
if redis.call("get",KEYS[1]) == ARGV[1] then
    return redis.call("del",KEYS[1])
else
    return 0
end`)
	res, _ := updateRecordExpireScript.Do(lock.Conn, lock.Key, lock.Value)
	fmt.Printf("RedisLock_AddTimeout_redis_Unlock lua res:%v\n", res)

	/*
		if lock.DoneExpireChan != nil{
			lock.Conn.Do("del", lock.Key)
		}
	*/
	if lock.DoneExpireChan != nil {
		close(lock.DoneExpireChan)
		lock.DoneExpireChan = nil
	}
	return
}

func (lock *RedisLock) ReExpire() {
	keyValueNow, err := redis.String(lock.Conn.Do("Get", lock.Key))
	if err != nil {
		fmt.Printf("RedisLock_AddTimeout_redis get Key fail :%v \n", err)
		return
	}
	if keyValueNow != lock.Value {
		close(lock.DoneExpireChan)
		lock.DoneExpireChan = nil
		fmt.Printf("RedisLock_AddTimeout_redis get Key success,but not expect value,now key is occupied by other :%v \n", keyValueNow)
		return
	}
	if lock.Expire.Seconds() < 1 {
		_, err = lock.Conn.Do("PEXPIRE", lock.Key, strconv.FormatInt(lock.Expire.Milliseconds(), 10))
	} else {
		_, err = lock.Conn.Do("EXPIRE", lock.Key, lock.Expire.Seconds())
	}
	if err != nil {
		fmt.Printf("RedisLock_AddTimeout_redis EXPIRE time fail :%v \n", err)
		return
	}
	fmt.Printf("RedisLock_AddTimeout_redis_Cron EXPIRE time success : %v\n", lock.Expire)
	return
}
