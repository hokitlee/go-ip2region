# Ip2region

## go mod

``` golang

go get -u github.com/hokitlee/go-ip2region/query

```

## 使用

``` golang
	// file download address https://github.com/hokitlee/go-ip2region/blob/master/data/ip2region.db
	dbFilePath := ""
	ip2, err := ip2region.New(dbFilePath)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if err := ip2.LoadToMemory(); err != nil {
		log.Fatalf("%v", err)
	}
	info, err := ip2.MemorySearch("192.168.0.1")
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("%v", info)
``` 