package main

import (
	"github.com/struqt/logging"
	"time"
)

func main() {
	logTest()
}

type e struct {
	str string
}

func (e e) Error() string {
	return e.str
}

func logTest() {
	logging.LogVerbosity = 2
	logging.LogRotateMBytes = 1
	logging.LogRotateFiles = 3
	logging.LogConsoleThreshold = -128
	log := logging.NewLogger("/tmp/demo.log").WithName("Example")

	{
		log := log.WithName("001").WithValues("module", "a")
		log.Error(e{str: "hello"}, "Logger in action!")
	}
	{
		log := log.WithName("002").WithValues("module", "b")
		log.Error(e{str: "hello"}, "Logger in action!")
	}

	for {
		log.V(0).Info("Logger in action!", "answer", 70)
		log.V(1).Info("Logger in action!", "answer", 71)
		log.V(2).Info("Logger in action!", "answer", 72)
		time.Sleep(500 * time.Millisecond)
	}
}
