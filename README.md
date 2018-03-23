openfigi
========

openfigi is a golang package providing access to the [OpenFIGI API]("https://openfigi.com/api)

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
