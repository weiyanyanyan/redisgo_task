package main

import (
	"github.com/docopt/docopt-go"

	"redisgo_task/config"
	"redisgo_task/task"
)

var usage string = `
redisgo_task

Usage:
	redisgo_task task --config=<file>
Options:
	-h --help     Show this screen.
	--version     Show version.
`

func main() {
	arguments, _ := docopt.ParseDoc(usage)
	if arguments["--config"] != nil {
		s := arguments["--config"].(string)
		if s == "" {
			panic("--config is inlivad")
		}
		config.LoadConfig(s)
	}

	if arguments["task"].(bool) {
		taskObj, err := task.New(config.GetConfig())
		if err != nil {
			return
		}
		err = taskObj.RunWithRedisLock()
		if err != nil {
			return
		}
	}
}
