# go-Ip2region

参考了项目[ip2region](https://github.com/lionsoul2014/ip2region),搜索算法与数据库结构基本一致，只不过在其中加入了区域以及isp编号，便于将ip归类。数据库生成代码原仓库是java实现的，这里使用go实现。支持通过多份数据生成数据库文件。可直接通过纯真数据库进行转换。

## 标准化的数据格式

每条ip数据段都固定了格式：

``` text
国家|区域|省份|城市|ISP|区域id|省份id|运营商id
```

## 使用

``` golang

go get -u github.com/hokitlee/go-ip2region/query

```

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