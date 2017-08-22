package report

import (
	"fmt"
	"os"
	"time"
)

type Result struct {
	StatusCode int
	Duration   time.Duration
}

type Report struct {
	average float64
	rps     float64

	n             int
	c             int
	reqNumTotal   int64
	reqNumFail    int64
	reqNumSucc    int64
	costTimeTotal float64
	results       chan *Result
}

func NewReport(n, c int) *Report {
	return &Report{
		n:       n,
		c:       c,
		results: make(chan *Result, n),
	}
}

func (self *Report) AddResult(res *Result) {
	self.results <- res
}

func (self *Report) Finalize(costTime float64) {
	self.costTimeTotal = costTime

	for {
		select {
		case res := <-self.results:
			self.reqNumTotal++
			if res.StatusCode != 200 {
				self.reqNumFail++
				fmt.Println("failed: ", res.StatusCode)
			} else {
				self.reqNumSucc++
			}

		default:
			self.rps = float64(self.reqNumTotal) / self.costTimeTotal
			self.average = self.costTimeTotal / float64(self.reqNumTotal)
			self.print()
			return
		}
	}
}

func (self *Report) print() {
	if self.reqNumTotal > 0 {
		fmt.Printf("Summary:\n")
		fmt.Printf(" Concurrency Level:\t%d\n", self.c)
		fmt.Printf(" Time taken for tests:\t%0.4f secs\n", self.costTimeTotal)
		fmt.Printf(" Complete requests:\t%d\n", self.reqNumTotal)
		fmt.Printf(" Failed requests:\t%d\n", self.reqNumFail)
		fmt.Printf(" Success requests:\t%d\n", self.reqNumSucc)
		fmt.Printf(" Requests per second:\t%0.4f\n", self.rps)
		fmt.Printf(" Average time per request:\t%0.4f\n", self.average)
	}

	os.Exit(0)
}
