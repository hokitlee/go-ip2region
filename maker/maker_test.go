package maker

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestMaker_make(t *testing.T) {
	rm := make(map[string]int)
	pm := make(map[string]int)
	im := make(map[string]int)
	rmf, err := os.Open("../data/area_code.csv")
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer rmf.Close()
	rd := bufio.NewReader(rmf)

	var province2Code []string
	for {
		bs, _, err := rd.ReadLine()
		if err == io.EOF {
			log.Println("read Province code file finish")
			break
		}
		if err != nil {
			t.Fatalf("%s", err)
		}
		province2Code = append(province2Code, string(bs))
	}

	for _, n := range province2Code {
		ss := strings.Split(n, ",")
		pId, err := strconv.Atoi(ss[1])
		if err != nil {
			t.Fatalf("%s", err)
		}
		rId, err := strconv.Atoi(ss[0])
		if err != nil {
			t.Fatalf("%s", err)
		}
		rm[ss[2]] = rId
		pm[ss[2]] = pId
	}
	isp := []string{"其他", "移动", "铁通", "联通", "电信", "内网IP"}
	ispId := []int{0, 1, 1, 2, 3, 4}
	for i := range isp {
		im[isp[i]] = ispId[i]
	}

	mf, err := os.Open("../data/ip.merge.txt")
	if err != nil {
		t.Fatalf("%s", err)
	}

	mfrd := bufio.NewReader(mf)
	var mds []Metadata

	for {
		bs, _, err := mfrd.ReadLine()
		if err == io.EOF {
			log.Println("read Province code file finish")
			break
		}
		if err != nil {
			t.Fatalf("%s", err)
		}

		ss := strings.Split(string(bs), "|")
		mds = append(mds, Metadata{
			StartIP:  ss[0],
			EndIP:    ss[1],
			Country:  ss[2],
			Province: ss[4],
			City:     ss[5],
			Isp:      ss[6],
		})
	}

	maker := NewMaker("./db.db", mds, rm, pm, im)
	err = maker.make()
	if err != nil {
		t.Fatalf("%s", err)
	}
}

func TestMergeMetadata(t *testing.T) {
	mf, err := os.Open("../data/ip.txt")
	if err != nil {
		t.Fatalf("%s", err)
	}
	mfrd := bufio.NewReader(mf)
	var mds []Metadata

	for {
		bs, _, err := mfrd.ReadLine()
		if err == io.EOF {
			log.Println("read Province code file finish")
			break
		}
		if err != nil {
			t.Fatalf("%s", err)
		}

		ss := strings.Split(string(bs), "|")
		mds = append(mds, Metadata{
			StartIP:  ss[0],
			EndIP:    ss[1],
			Country:  ss[2],
			Province: ss[4],
			City:     ss[5],
			Isp:      ss[6],
		})
	}

	mergeMetaData := make([]Metadata, 0)

	mergeMetaData = append(mergeMetaData, Metadata{
		StartIP:  "0.0.0.0",
		EndIP:    "1.0.0.255",
		Country:  "test",
		Province: "test",
		City:     "test",
		Isp:      "test",
	})

	ms := MergeMetadata(mds, mergeMetaData)

	for _, n := range ms {
		fmt.Printf("%-10v \n", n)
	}

}
