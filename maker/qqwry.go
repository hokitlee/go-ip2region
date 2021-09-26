package maker

import (
	"encoding/binary"
	"fmt"
	"github.com/yanyiwu/gojieba"
	"github.com/yinheli/mahonia"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	INDEX_LEN       = 7
	REDIRECT_MODE_1 = 0x01
	REDIRECT_MODE_2 = 0x02
)

type IpInfo struct {
	Ip      string
	Country string
	City    string
}

func (ip *IpInfo) String() string {
	return fmt.Sprintf("ip: %s, country: %s, city: %s", ip.Ip, ip.Country, ip.City)
}

// @author yinheli
type QQwry struct {
	Ip       string
	Country  string
	City     string
	filepath string
	file     *os.File
}

func NewQQwry(file string) (qqwry *QQwry) {
	qqwry = &QQwry{filepath: file}
	return
}

func (qw *QQwry) Find(ip string) {
	if qw.filepath == "" {
		return
	}

	file, err := os.OpenFile(qw.filepath, os.O_RDONLY, 0400)
	defer file.Close()
	if err != nil {
		return
	}
	qw.file = file

	qw.Ip = ip
	offset := qw.searchIndex(binary.BigEndian.Uint32(net.ParseIP(ip).To4()))
	// log.Println("loc offset:", offset)
	if offset <= 0 {
		return
	}

	var country []byte
	var area []byte

	mode := qw.readMode(offset + 4)
	// log.Println("mode", mode)
	if mode == REDIRECT_MODE_1 {
		countryOffset := qw.readUInt24()
		mode = qw.readMode(countryOffset)
		// log.Println("1 - mode", mode)
		if mode == REDIRECT_MODE_2 {
			c := qw.readUInt24()
			country = qw.readString(c)
			countryOffset += 4
		} else {
			country = qw.readString(countryOffset)
			countryOffset += uint32(len(country) + 1)
		}
		area = qw.readArea(countryOffset)
	} else if mode == REDIRECT_MODE_2 {
		countryOffset := qw.readUInt24()
		country = qw.readString(countryOffset)
		area = qw.readArea(offset + 8)
	} else {
		country = qw.readString(offset + 4)
		area = qw.readArea(offset + uint32(5+len(country)))
	}

	enc := mahonia.NewDecoder("gbk")
	qw.Country = enc.ConvertString(string(country))
	qw.City = enc.ConvertString(string(area))

}

func (qw *QQwry) Iterate(ch chan IpInfo) {
	go func() {
		file, err := os.OpenFile(qw.filepath, os.O_RDONLY, 0755)
		defer file.Close()
		if err != nil {
			return
		}
		qw.file = file

		defer close(ch)

		header := make([]byte, 8)
		qw.file.Seek(0, 0)
		qw.file.Read(header)

		start := binary.LittleEndian.Uint32(header[:4])
		end := binary.LittleEndian.Uint32(header[4:])

		for s := start; s <= end-7; s += 7 {
			info := qw.readOneRecord(s)
			ch <- info
		}
	}()
}

func (qw *QQwry) readOneRecord(offset uint32) IpInfo {
	qw.file.Seek(int64(offset), 0)
	buf := make([]byte, INDEX_LEN)
	qw.file.Read(buf)
	_ip := binary.LittleEndian.Uint32(buf[:4])
	recordAddr := byte3ToUInt32(buf[4:])

	var country []byte
	var area []byte

	mode := qw.readMode(recordAddr + 4)
	// log.Println("mode", mode)
	if mode == REDIRECT_MODE_1 {
		countryOffset := qw.readUInt24()
		mode = qw.readMode(countryOffset)
		// log.Println("1 - mode", mode)
		if mode == REDIRECT_MODE_2 {
			c := qw.readUInt24()
			country = qw.readString(c)
			countryOffset += 4
		} else {
			country = qw.readString(countryOffset)
			countryOffset += uint32(len(country) + 1)
		}
		area = qw.readArea(countryOffset)
	} else if mode == REDIRECT_MODE_2 {
		countryOffset := qw.readUInt24()
		country = qw.readString(countryOffset)
		area = qw.readArea(recordAddr + 8)
	} else {
		country = qw.readString(recordAddr + 4)
		area = qw.readArea(recordAddr + uint32(5+len(country)))
	}
	ipb := make([]byte, 4)
	binary.BigEndian.PutUint32(ipb, _ip)
	enc := mahonia.NewDecoder("gbk")

	info := IpInfo{
		Ip:      byte4ToIpString(ipb),
		Country: enc.ConvertString(string(country)),
		City:    enc.ConvertString(string(area)),
	}

	return info
}

func byte4ToIpString(b []byte) string {
	res := strconv.Itoa(int(b[0]))
	for i := 1; i < len(b); i++ {
		res += "." + strconv.Itoa(int(b[i]))
	}
	return res
}

func (qw *QQwry) readMode(offset uint32) byte {
	qw.file.Seek(int64(offset), 0)
	mode := make([]byte, 1)
	qw.file.Read(mode)
	return mode[0]
}

func (qw *QQwry) readArea(offset uint32) []byte {
	mode := qw.readMode(offset)
	if mode == REDIRECT_MODE_1 || mode == REDIRECT_MODE_2 {
		areaOffset := qw.readUInt24()
		if areaOffset == 0 {
			return []byte("")
		} else {
			return qw.readString(areaOffset)
		}
	} else {
		return qw.readString(offset)
	}
	return []byte("")
}

func (qw *QQwry) readString(offset uint32) []byte {
	qw.file.Seek(int64(offset), 0)
	data := make([]byte, 0, 30)
	buf := make([]byte, 1)
	for {
		qw.file.Read(buf)
		if buf[0] == 0 {
			break
		}
		data = append(data, buf[0])
	}
	return data
}

func (qw *QQwry) searchIndex(ip uint32) uint32 {
	header := make([]byte, 8)
	qw.file.Seek(0, 0)
	qw.file.Read(header)

	start := binary.LittleEndian.Uint32(header[:4])
	end := binary.LittleEndian.Uint32(header[4:])

	// log.Printf("len info %v, %v ---- %v, %v", start, end, hex.EncodeToString(header[:4]), hex.EncodeToString(header[4:]))

	for {
		mid := qw.getMiddleOffset(start, end)
		qw.file.Seek(int64(mid), 0)
		buf := make([]byte, INDEX_LEN)
		qw.file.Read(buf)
		_ip := binary.LittleEndian.Uint32(buf[:4])

		// log.Printf(">> %v, %v, %v -- %v", start, mid, end, hex.EncodeToString(buf[:4]))

		if end-start == INDEX_LEN {
			offset := byte3ToUInt32(buf[4:])
			qw.file.Read(buf)
			if ip < binary.LittleEndian.Uint32(buf[:4]) {
				return offset
			} else {
				return 0
			}
		}

		// 找到的比较大，向前移
		if _ip > ip {
			end = mid
		} else if _ip < ip { // 找到的比较小，向后移
			start = mid
		} else if _ip == ip {
			return byte3ToUInt32(buf[4:])
		}

	}
	return 0
}

func (qw *QQwry) readUInt24() uint32 {
	buf := make([]byte, 3)
	qw.file.Read(buf)
	return byte3ToUInt32(buf)
}

func (qw *QQwry) getMiddleOffset(start uint32, end uint32) uint32 {
	records := ((end - start) / INDEX_LEN) >> 1
	return start + records*INDEX_LEN
}

func byte3ToUInt32(data []byte) uint32 {
	i := uint32(data[0]) & 0xff
	i |= (uint32(data[1]) << 8) & 0xff00
	i |= (uint32(data[2]) << 16) & 0xff0000
	return i
}

func (qw *QQwry) GetQQWryIpRecord(pm map[string]int) ([]Metadata, error) {
	log.Println("start use qqwry db get metadata")
	ipInfos := make([]IpInfo, 0)
	jieba := gojieba.NewJieba()
	ch := make(chan IpInfo, 100)

	qw.Iterate(ch)
	for n := range ch {
		ipInfos = append(ipInfos, n)
	}

	ipRcs := make([]Metadata, 0)

	for i := 0; i < len(ipInfos)-1; i++ {
		n := ipInfos[i]
		eIpN, err := Ip2long(ipInfos[i+1].Ip)
		if err != nil {
			return nil, err
		}
		r := Metadata{
			StartIP: n.Ip,
			EndIP:   IpLong2String(eIpN - 1),
		}
		cs := jieba.Cut(n.Country, true)
		for i, n := range cs {
			switch i {
			case 0:
				s := strings.TrimRight(strings.TrimRight(n, "市"), "省")
				if _, ok := pm[s]; ok {
					r.Province = s
					r.Country = "中国"
				} else {
					r.Country = n
				}
			case 1:
				r.City = n
			}
		}
		r.Isp = n.City
		r.Format()
		ipRcs = append(ipRcs, r)
	}
	log.Println("use qqwry db get metadata finish")

	return ipRcs, nil
}
