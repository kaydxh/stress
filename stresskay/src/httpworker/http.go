package httpworker

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"report"
	"strings"
	"sync"
	"time"
)

type HeaderSlice []string

func (h *HeaderSlice) String() string {
	return fmt.Sprintf("%s", *h)
}

func (h *HeaderSlice) Set(value string) error {
	*h = append(*h, value)
	return nil
}

type RequestConf struct {
	Weight      int    `weight`
	Method      string `method`
	Url         string `url`
	ContentType string `contenttype`
	PostData    string `postdata`
}

type HttpConf struct {
	KeepAlive bool     `keepalive`
	Header    []string `header`
	Timeout   int      `timeout`
	Request   []RequestConf
}

type HttpWorker struct {
	n        int
	c        int
	weight   int
	httpConf *HttpConf

	requests []*http.Request
	report   *report.Report
}

func NewHttpConf(keepalive bool, header []string, t, reqN int) *HttpConf {
	return &HttpConf{
		KeepAlive: keepalive,
		Header:    header,
		Timeout:   t,
		Request:   make([]RequestConf, reqN),
	}
}

func NewHttpWorker(n, c int, httpConf *HttpConf) *HttpWorker {
	return &HttpWorker{
		n:        n,
		c:        c,
		httpConf: httpConf,
		requests: make([]*http.Request, 0),
		report:   report.NewReport(n, c),
	}
}

func (self *HttpWorker) MakeRequest() error {
	var header []string

	type request struct {
		preq *http.Request
		conf *RequestConf
	}

	reqs := make([]*request, 0)

	for _, reqC := range self.httpConf.Request {
		method := strings.ToUpper(reqC.Method)
		if method == "" {
			continue
		}
		if method != "GET" && method != "POST" {
			return errors.New("no support this method")
		}

		r, err := http.NewRequest(method, reqC.Url, nil)
		if err != nil {
			return err
		}

		if len(reqC.ContentType) != 0 {
			r.Header.Set("Content-Type", reqC.ContentType)
		}

		for _, h := range self.httpConf.Header {
			header = strings.SplitN(h, ":", 2)
			if len(header) != 2 {
				return errors.New("invaild http header")
			}

			r.Header.Add(header[0], header[1])
		}

		for i := 0; i < reqC.Weight; i++ {
			reqs = append(reqs, &request{preq: r, conf: &reqC})
		}

		self.weight += reqC.Weight
	}

	//fmt.Printf("weight:%d\n", self.weight)
	if self.weight == 0 {
		self.weight = 1
	}

	for i := 0; i < self.n; i++ {
		req := reqs[i%self.weight]
		self.requests = append(self.requests, cloneRequest(req.preq, req.conf.PostData))
	}

	return nil
}

func (self *HttpWorker) Run() {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)

	fmt.Println("start...")
	st := time.Now()
	go func() {
		<-s
		fmt.Println("receive sigint")
		self.report.Finalize(time.Now().Sub(st).Seconds())
		os.Exit(1)
	}()

	self.startWorkers()
	self.report.Finalize(time.Now().Sub(st).Seconds())

}

func (self *HttpWorker) startWorkers() {
	var wg sync.WaitGroup
	wg.Add(self.c)

	for i := 0; i < self.c; i++ {
		go func(rid int) {
			self.startWorker(rid, self.n/self.c)
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func (self *HttpWorker) startWorker(rid, num int) {
	tr := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   time.Duration(self.httpConf.Timeout) * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		DisableKeepAlives: !self.httpConf.KeepAlive,
	}

	hc := &http.Client{Transport: tr}

	for i := 0; i < num; i++ {
		req := self.requests[rid*num+i]
		self.sendRequest(hc, req)
	}
}

func (self *HttpWorker) sendRequest(hc *http.Client, req *http.Request) {
	s := time.Now()
	var code int

	resp, err := hc.Do(req)
	if err == nil {
		code = resp.StatusCode
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	} else {
		fmt.Println("Do err: ", err)
	}

	self.report.AddResult(&report.Result{
		StatusCode: code,
		Duration:   time.Now().Sub(s),
	})
}

func cloneRequest(r *http.Request, body string) *http.Request {
	r2 := new(http.Request)
	*r2 = *r

	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	if len(body) > 0 {
		r2.Body = ioutil.NopCloser(strings.NewReader(body))
	}

	return r2

}
