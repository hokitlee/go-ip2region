package maker

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

const regionRecordMeteLineSize = 3

// format: 1,11,北京
func RegionMap(path string) (map[string]int,error) {

	rmf, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer rmf.Close()
	rd := bufio.NewReader(rmf)

	m := make(map[string]int)
	for {
		line,_,err := rd.ReadLine()
		if err == io.EOF {
			log.Println("read file finish")
			break
		}
		if err != nil {
			return nil, err
		}
		s := strings.Split(string(line),",")
		if len(s) != regionRecordMeteLineSize {
			return nil, errors.New("region file format err")
		}

		code,err := strconv.Atoi(s[0])
		if err != nil {
			return nil,err
		}
		m[s[2]] = code
	}
	return m,nil
}


// format: 1,11,北京
func ProvinceMap(path string) (map[string]int,error) {

	rmf, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer rmf.Close()
	rd := bufio.NewReader(rmf)

	m := make(map[string]int)
	for {
		line,_,err := rd.ReadLine()
		if err == io.EOF {
			log.Println("read file finish")
			break
		}
		if err != nil {
			return nil, err
		}
		s := strings.Split(string(line),",")
		if len(s) != regionRecordMeteLineSize {
			return nil, errors.New("region file format err")
		}

		code,err := strconv.Atoi(s[1])
		if err != nil {
			return nil,err
		}
		m[s[2]] = code
	}
	return m,nil
}

// format: 1,1,移动
func IspMap(path string) (map[string]int, error) {
	rmf, err := os.Open(path)

	if err != nil {
		return nil, err
	}
	defer rmf.Close()
	rd := bufio.NewReader(rmf)

	m := make(map[string]int)

	for {
		line,_,err := rd.ReadLine()
		if err == io.EOF {
			log.Println("read file finish")
			break
		}
		if err != nil {
			return nil, err
		}
		s := strings.Split(string(line),",")
		if len(s) != regionRecordMeteLineSize {
			return nil, errors.New("region file format err")
		}

		code,err := strconv.Atoi(s[1])
		if err != nil {
			return nil,err
		}
		m[s[2]] = code
	}
	return m,nil
}