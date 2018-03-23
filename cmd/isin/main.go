package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/tomarus/openfigi"
)

func main() {
	// Enable this to enable Redis caching.
	// RedisCache("192.168.0.3:6379")

	e := flag.String("e", "", "Limit queries to this exchange.") // e.g. "US"
	flag.Parse()
	a := flag.Args()

	if len(a) == 0 {
		fmt.Printf("Usage: $v isincode\n", os.Args[0])
		os.Exit(1)
	}

	req, err := openfigi.NewRequest("ID_ISIN", a[0])
	if err != nil {
		log.Fatal(err)
	}

	if *e != "" {
		req.Exchange(*e)
	}

	res, err := req.Do()
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(res)
}
