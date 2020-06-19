package ip2region

import (
	"log"
	"math/rand"
	"os"
	"testing"
	"time"
)

const IpDbAddress = "../data/ip2region.db"

var ipr *Ip2Region

func TestMain(m *testing.M) {
	log.Print("ini test")
	var err error
	ipr, err = New(IpDbAddress)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func BenchmarkBtreeSearch(B *testing.B) {
	region, err := New(IpDbAddress)
	if err != nil {
		B.Error(err)
	}
	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		region.BtreeSearch("127.0.0.1")
	}
}

func BenchmarkMemorySearch(B *testing.B) {
	region, err := New(IpDbAddress)
	if err != nil {
		B.Error(err)
	}
	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		region.MemorySearch("127.0.0.1")
	}
	//fmt.Println("-------------" + time.Since(t).String())
}

func BenchmarkBinarySearch(B *testing.B) {
	region, err := New(IpDbAddress)
	if err != nil {
		B.Error(err)
	}
	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		region.BinarySearch("127.0.0.1")
	}
}

func TestIp2Region_MemorySearch(t *testing.T) {
	var err error
	ipr, err = New(IpDbAddress)
	if err != nil {
		log.Fatal(err)
	}
	err = ipr.LoadToMemory()
	if err != nil {
		t.Fatalf("%v", err)
	}
	ipStr := "255.255.255.255"

	ipNum, _ := Ip2long(ipStr)
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100; i++ {

		ip := IpLong2String(rand.Int63n(ipNum))

		info, err := ipr.MemorySearch(ip)
		t.Logf("ip:%20s :%-20s\n", ip, info.String())
		if err != nil {
			t.Fatalf("%s", err)
		}

	}
}
