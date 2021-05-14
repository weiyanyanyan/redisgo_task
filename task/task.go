package task

import (
	"fmt"
	"time"

	"redisgo_task/config"
	"redisgo_task/util/lock"
)

type Task struct {
	cfg *config.Config
}

func New(cfg *config.Config) (*Task, error) {
	return &Task{
		cfg: cfg,
	}, nil
}

func (c *Task) RunWithRedisLock() error {
	fmt.Printf("Task_RunWithRedisLock Locking\n")
	var lockObj, err = lock.NewRedisLock(&c.cfg.RedisInfo)
	if err != nil {
		return err
	}
	lockObj.Lock()
	defer lockObj.Unlock()
	fmt.Printf("Task_RunWithRedisLock Locked\n")
	//todo task
	time.Sleep(3 * time.Second)
	return err
}

