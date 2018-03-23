OpenFIGI
========

OpenFIGI is a golang package providing access to the [OpenFIGI API](https://openfigi.com/api)

### What is OpenFIGI

OpenFIGI is your entry point to multiple tools for identifying, mapping and requesting a free and open symbology dataset. This user friendly platform provides the ultimate understanding for how a unique identifier combined with accurate, associated metadata can eliminate redundant mapping processes, streamline the trade workflow and reduce operational risk

Please see https://openfigi.com/ for all details.

### Usage

```
go get github.com/tomarus/openfigi
```

Example:

``` go
	import "github.com/tomarus/openfigi"

	req, err := openfigi.NewRequest("ID_ISIN", "US3623931009")
	if err != nil {
		log.Fatal(err)
	}

	req.Exchange("US")

	res, err := req.Do()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%+v", res)
```

### Feedback

This is an experimental package unaffiliated with the [OpenFIGI](https://openfigi.com) project.