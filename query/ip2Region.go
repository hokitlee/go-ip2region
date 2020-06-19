package ip2region

import (
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const (
	IndexBlockLength  = 12
	TotalHeaderLength = 8192
)

type IpInfo struct {
	Country    string
	Province   string
	City       string
	ISP        string
	RegionId   int64
	ProvinceId int64
	ISPId      int64
}

func (ip IpInfo) String() string {
	return ip.Country + "|" + ip.Province + "|" + ip.City + "|" + ip.ISP + "|" +
		strconv.FormatInt(ip.RegionId, 10) + "|" + strconv.FormatInt(ip.ProvinceId, 10) + "|" +
		strconv.FormatInt(ip.ISPId, 10)
}

type Ip2Region struct {
	// db file handler
	dbFileHandler *os.File

	//header block info

	headerSip []int64
	headerPtr []int64
	headerLen int64

	// super block index info
	firstIndexPtr int64
	lastIndexPtr  int64
	totalBlocks   int64

	// for memory mode only
	// the original db binary string

	dbBinStr []byte
	dbFile   string
}

func New(path string) (*Ip2Region, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &Ip2Region{
		dbFile:        path,
		dbFileHandler: file,
	}, nil
}

func (ipr *Ip2Region) Close() error {
	return ipr.dbFileHandler.Close()
}

func (ipr *Ip2Region) loadToMemory() error {
	var err error
	if ipr.totalBlocks == 0 {
		ipr.dbBinStr, err = ioutil.ReadFile(ipr.dbFile)

		if err != nil {
			return err
		}

		ipr.firstIndexPtr = GetLong(ipr.dbBinStr, 0)
		ipr.lastIndexPtr = GetLong(ipr.dbBinStr, 4)
		ipr.totalBlocks = (ipr.lastIndexPtr-ipr.firstIndexPtr)/IndexBlockLength + 1
	}
	return err
}

// Notion: Need to call a before loadToMemory
func (ipr *Ip2Region) MemorySearch(ipStr string) (ipInfo IpInfo, err error) {
	ipInfo = IpInfo{}

	if ipr.totalBlocks == 0 {
		ipr.dbBinStr, err = ioutil.ReadFile(ipr.dbFile)

		if err != nil {

			return ipInfo, err
		}

		ipr.firstIndexPtr = GetLong(ipr.dbBinStr, 0)
		ipr.lastIndexPtr = GetLong(ipr.dbBinStr, 4)
		ipr.totalBlocks = (ipr.lastIndexPtr-ipr.firstIndexPtr)/IndexBlockLength + 1
	}

	ip, err := Ip2long(ipStr)
	if err != nil {
		return ipInfo, err
	}

	h := ipr.totalBlocks
	var dataPtr, l int64
	for l <= h {

		m := (l + h) >> 1
		p := ipr.firstIndexPtr + m*IndexBlockLength
		sip := GetLong(ipr.dbBinStr, p)
		if ip < sip {
			h = m - 1
		} else {
			eip := GetLong(ipr.dbBinStr, p+4)
			if ip > eip {
				l = m + 1
			} else {
				dataPtr = GetLong(ipr.dbBinStr, p+8)
				break
			}
		}
	}
	if dataPtr == 0 {
		return ipInfo, errors.New("not found")
	}

	dataLen := (dataPtr >> 24) & 0xFF
	dataPtr = dataPtr & 0x00FFFFFF
	ipInfo = getIpInfo(ipr.dbBinStr[(dataPtr) : dataPtr+dataLen])
	return ipInfo, nil
}

func (ipr *Ip2Region) BinarySearch(ipStr string) (ipInfo IpInfo, err error) {
	ipInfo = IpInfo{}
	if ipr.totalBlocks == 0 {
		ipr.dbFileHandler.Seek(0, 0)
		superBlock := make([]byte, 8)
		ipr.dbFileHandler.Read(superBlock)
		ipr.firstIndexPtr = GetLong(superBlock, 0)
		ipr.lastIndexPtr = GetLong(superBlock, 4)
		ipr.totalBlocks = (ipr.lastIndexPtr-ipr.firstIndexPtr)/IndexBlockLength + 1
	}

	var l, dataPtr, p int64

	h := ipr.totalBlocks

	ip, err := Ip2long(ipStr)

	if err != nil {
		return
	}

	for l <= h {
		m := (l + h) >> 1

		p = m * IndexBlockLength

		_, err = ipr.dbFileHandler.Seek(ipr.firstIndexPtr+p, 0)
		if err != nil {
			return
		}

		buffer := make([]byte, IndexBlockLength)
		_, err = ipr.dbFileHandler.Read(buffer)

		if err != nil {

		}
		sip := GetLong(buffer, 0)
		if ip < sip {
			h = m - 1
		} else {
			eip := GetLong(buffer, 4)
			if ip > eip {
				l = m + 1
			} else {
				dataPtr = GetLong(buffer, 8)
				break
			}
		}
	}

	if dataPtr == 0 {
		err = errors.New("not found")
		return
	}

	dataLen := (dataPtr >> 24) & 0xFF
	dataPtr = dataPtr & 0x00FFFFFF

	ipr.dbFileHandler.Seek(dataPtr, 0)
	data := make([]byte, dataLen)
	ipr.dbFileHandler.Read(data)
	ipInfo = getIpInfo(data[4:dataLen])
	err = nil
	return
}

func (ipr *Ip2Region) BtreeSearch(ipStr string) (ipInfo IpInfo, err error) {
	ipInfo = IpInfo{}
	ip, err := Ip2long(ipStr)

	if ipr.headerLen == 0 {
		ipr.dbFileHandler.Seek(8, 0)

		buffer := make([]byte, TotalHeaderLength)
		ipr.dbFileHandler.Read(buffer)
		var idx int64
		for i := 0; i < TotalHeaderLength; i += 8 {
			startIp := GetLong(buffer, int64(i))
			dataPar := GetLong(buffer, int64(i+4))
			if dataPar == 0 {
				break
			}

			ipr.headerSip = append(ipr.headerSip, startIp)
			ipr.headerPtr = append(ipr.headerPtr, dataPar)
			idx++
		}

		ipr.headerLen = idx
	}

	var l, sptr, eptr int64
	h := ipr.headerLen

	for l <= h {
		m := int64(l+h) >> 1
		if m < ipr.headerLen {
			if ip == ipr.headerSip[m] {
				if m > 0 {
					sptr = ipr.headerPtr[m-1]
					eptr = ipr.headerPtr[m]
				} else {
					sptr = ipr.headerPtr[m]
					eptr = ipr.headerPtr[m+1]
				}
				break
			}
			if ip < ipr.headerSip[m] {
				if m == 0 {
					sptr = ipr.headerPtr[m]
					eptr = ipr.headerPtr[m+1]
					break
				} else if ip > ipr.headerSip[m-1] {
					sptr = ipr.headerPtr[m-1]
					eptr = ipr.headerPtr[m]
					break
				}
				h = m - 1
			} else {
				if m == ipr.headerLen-1 {
					sptr = ipr.headerPtr[m-1]
					eptr = ipr.headerPtr[m]
					break
				} else if ip <= ipr.headerSip[m+1] {
					sptr = ipr.headerPtr[m]
					eptr = ipr.headerPtr[m+1]
					break
				}
				l = m + 1
			}
		}

	}

	if sptr == 0 {
		err = errors.New("not found")
		return
	}

	blockLen := eptr - sptr
	ipr.dbFileHandler.Seek(sptr, 0)
	index := make([]byte, blockLen+IndexBlockLength)
	ipr.dbFileHandler.Read(index)
	var dataptr int64
	h = blockLen / IndexBlockLength
	l = 0

	for l <= h {
		m := int64(l+h) >> 1
		p := m * IndexBlockLength
		sip := GetLong(index, p)
		if ip < sip {
			h = m - 1
		} else {
			eip := GetLong(index, p+4)
			if ip > eip {
				l = m + 1
			} else {
				dataptr = GetLong(index, p+8)
				break
			}
		}
	}

	if dataptr == 0 {
		err = errors.New("not found")
		return
	}

	dataLen := (dataptr >> 24) & 0xFF
	dataPtr := dataptr & 0x00FFFFFF

	ipr.dbFileHandler.Seek(dataPtr, 0)
	data := make([]byte, dataLen)
	ipr.dbFileHandler.Read(data)
	ipInfo = getIpInfo(data[4:])
	return
}

func GetLong(b []byte, offset int64) int64 {
	val := int64(b[offset]) |
		int64(b[offset+1])<<8 |
		int64(b[offset+2])<<16 |
		int64(b[offset+3])<<24

	return val

}

func writeIntLong(b []byte, offset int, v int64) {
	b[offset] = byte((v >> 0) & 0xFF)
	offset++
	b[offset] = byte((v >> 8) & 0xFF)
	offset++
	b[offset] = byte((v >> 16) & 0xFF)
	offset++
	b[offset] = byte((v >> 24) & 0xFF)
}

func Ip2long(IpStr string) (int64, error) {
	bits := strings.Split(IpStr, ".")
	if len(bits) != 4 {
		return 0, errors.New("ip format error")
	}

	var sum int64
	for i, n := range bits {
		bit, _ := strconv.ParseInt(n, 10, 64)
		sum += bit << uint(24-8*i)
	}

	return sum, nil
}

func IpLong2String(n int64) string {
	var m int64 = 8
	s := make([]string, 4)
	for i := 3; i >= 0; n >>= m {
		s[i] = strconv.Itoa(int(n & ((1 << m) - 1)))
		i--
	}
	ipStr := strings.Join(s, ".")
	return ipStr
}

func getIpInfo(line []byte) IpInfo {

	lineSlice := strings.Split(string(line), "|")
	ipInfo := IpInfo{}
	length := len(lineSlice)
	//ipInfo.RegionId = regionId
	if length < 6 {
		for i := 0; i <= 6-length; i++ {
			lineSlice = append(lineSlice, "")
		}
	}
	rId, err := strconv.Atoi(lineSlice[4])
	if err != nil {
		rId = 0
	}
	pId, err := strconv.Atoi(lineSlice[5])
	if err != nil {
		pId = 0
	}
	sId, err := strconv.Atoi(lineSlice[6])
	if err != nil {
		sId = 0
	}

	ipInfo.Country = lineSlice[0]
	ipInfo.Province = lineSlice[1]
	ipInfo.City = lineSlice[2]
	ipInfo.ISP = lineSlice[3]
	ipInfo.RegionId = int64(rId)
	ipInfo.ProvinceId = int64(pId)
	ipInfo.ISPId = int64(sId)
	return ipInfo
}
