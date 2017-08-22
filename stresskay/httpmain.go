package main

import (
	"conf"
	"flag"
	"fmt"
	"httpworker"
	"os"
	"runtime"
)

var (
	n, c, t     int
	keepalive   bool
	url         string
	reqBody     string
	contentType string
	header      httpworker.HeaderSlice

	cfgFile string
)

var usage = `Usage: httpTest [options...] <url> or -f configfile

options:
    -n Number of requests to run. default: 200.
    -c Number of request to run concurrency. default: 50.
    -t Request connection timeout in second. default: 30s.
    -H Custom Http header, eg. -H "Accept: text/html" -H "Content-Type: application/xml".
    -k[=true|false] Http keep-alive. default: false.
    -d Http request body to POST.
    -T Content-type header to POST, eg. 'application/x-www-form-urlencode'. default: text/plain.
`

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

	flag.IntVar(&n, "n", 200, "")
	flag.IntVar(&c, "c", 50, "")
	flag.IntVar(&t, "t", 30, "")
	flag.BoolVar(&keepalive, "k", false, "")
	flag.StringVar(&cfgFile, "f", "./httpconfig", "")
	flag.StringVar(&reqBody, "d", "", "")
	flag.StringVar(&contentType, "T", "text/plain", "")
	flag.Var(&header, "H", "")
	flag.Parse()

	if flag.NArg() < 1 && cfgFile == "" {
		abort("missing params.")
	}

	method := "GET"
	if reqBody != "" {
		method = "POST"
	}

	var (
		secN     int = 10
		httpConf httpworker.HttpConf
	)

	if flag.NArg() > 0 {
		httpConf = *httpworker.NewHttpConf(keepalive, header, t, secN)
		url = flag.Args()[0]
		httpConf.Request[0] = httpworker.RequestConf{
			Weight:      1,
			Method:      method,
			Url:         url,
			ContentType: contentType,
			PostData:    reqBody,
		}
	} else {
		cfg := conf.NewConf()
		err := cfg.LoadFile(cfgFile)
		if err != nil {
			abort(err.Error())
		}

		secN = cfg.GetSectionNum()

		//fmt.Println("secN: ", secN)
		httpConf = *httpworker.NewHttpConf(keepalive, header, t, secN)

		err = cfg.Parse(&httpConf)
		if err != nil {
			abort(err.Error())
		}

	}

	worker := httpworker.NewHttpWorker(n, c, &httpConf)

	//fmt.Println("httpConf: ", httpConf)
	err := worker.MakeRequest()
	if err != nil {
		abort(err.Error())
	}

	worker.Run()

}

func abort(errmsg string) {
	if errmsg != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", errmsg)
	}

	flag.Usage()
	os.Exit(1)
}
