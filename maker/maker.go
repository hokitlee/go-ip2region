package maker

import (
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

/**
 * fast ip db maker
 * <p>
 * db struct:
 * 1. header part
 * 1): super part:
 * +------------+-----------+
 * | 4 bytes	| 4 bytes   |
 * +------------+-----------+
 * start index ptr, end index ptr
 * <p>
 * 2): b-tree index part
 * +------------+-----------+-----------+-----------+
 * | 4bytes		| 4bytes	| 4bytes	| 4bytes	| ...
 * +------------+-----------+-----------+-----------+
 * start ip ptr  index ptr
 * <p>
 * 2. data part:
 * +------------+-----------------------+
 * | 2bytes		| dynamic length 		|
 * +------------+-----------------------+
 * data length   city_id|Country|Province|Area|City|ISP
 * <p>
 * 3. index part: (ip range)
 * +------------+-----------+---------------+
 * | 4bytes		| 4bytes	| 4bytes		|
 * +------------+-----------+---------------+
 * start ip 	  end ip	  3 byte data ptr & 1 byte data length
 *
 */

type DateBlock struct {
	country    string
	province   string
	city       string
	isp        string
	regionId   int
	provinceId int
	ispId      int
}

func (dbl *DateBlock) Bytes() []byte {
	return []byte(dbl.String())
}

func (dbl *DateBlock) String() string {
	var s []string
	if dbl.country == "" {
		s = append(s, "0")
	} else {
		s = append(s, dbl.country)
	}
	if dbl.province == "" {
		s = append(s, "0")
	} else {
		s = append(s, dbl.province)
	}
	if dbl.city == "" {
		s = append(s, "0")
	} else {
		s = append(s, dbl.city)
	}
	if dbl.isp == "" {
		s = append(s, "0")
	} else {
		s = append(s, dbl.isp)
	}
	s = append(s, strconv.Itoa(dbl.regionId), strconv.Itoa(dbl.provinceId), strconv.Itoa(dbl.ispId))
	return strings.Join(s, "|")
}

type Metadata struct {
	StartIP  string
	EndIP    string
	Country  string
	Province string
	City     string
	Isp      string
}

func (md *Metadata) String() string {
	return md.StartIP + "|" + md.EndIP + "|" + md.Country + "|" + md.Province + "|" + md.City + "|" + md.Isp
}

func (md *Metadata) RegionString() string {
	return md.Country + "|" + md.Province + "|" + md.City + "|" + md.Isp
}

func (md *Metadata) Format() {
	if md.Country == "" {
		md.Country = "0"
	}
	if md.Province == "" {
		md.Province = "0"
	}
	if md.City == "" {
		md.City = "0"
	}
	if md.Isp == "" {
		md.Isp = "0"
	}
}

func (md *Metadata) toDateBlock() DateBlock {
	return DateBlock{
		country:  md.Country,
		province: md.Province,
		city:     md.City,
		isp:      md.Isp,
	}
}

type IndexBlock struct {
	//12
	startIP int64
	endIP   int64
	dataPtr int
	dataLen int
}

func (ib *IndexBlock) getBytes() []byte {
	/*
	 * +------------+-----------+-----------+
	 * | 4bytes        | 4bytes    | 4bytes    |
	 * +------------+-----------+-----------+
	 *  start ip      end ip      data ptr + len
	 */
	b := make([]byte, 12)
	WriteIntLong(b, 0, ib.startIP)
	WriteIntLong(b, 4, ib.endIP)
	mix := ib.dataPtr | ((ib.dataLen << 24) & 0xFF000000)
	WriteIntLong(b, 8, int64(mix))
	return b
}

type HeaderBlock struct {

	/**
	 * index block start ip address
	 */
	indexStartIp int64

	/**
	 * ip address
	 */
	indexPtr int
}

func (db *HeaderBlock) Bytes() []byte {

	b := make([]byte, 8)

	WriteIntLong(b, 0, db.indexStartIp)
	WriteIntLong(b, 4, int64(db.indexPtr))

	return b
}

type Maker struct {
	dbFilePath string

	dbFile *os.File
	// 8*2048
	totalHeaderSize int
	// 4*2048
	indexBlockSize int

	// 12
	indexBlockLength int

	indexPool []IndexBlock

	headerBlockPool []HeaderBlock

	metadata []Metadata

	regionCodeMap map[string]int

	provinceCodeMap map[string]int

	ispCodeMap map[string]int

	regionRecordMap map[string]IndexBlock
}

func NewMaker(dbFilePath string, md []Metadata, rm, pm, im map[string]int) *Maker {
	if rm == nil {
		rm = make(map[string]int)
	}
	if pm == nil {
		pm = make(map[string]int)
	}
	if im == nil {
		im = make(map[string]int)
	}

	mk := &Maker{
		dbFilePath:       dbFilePath,
		dbFile:           nil,
		totalHeaderSize:  8 * 2048,
		indexBlockSize:   4 * 2048,
		indexBlockLength: 12,
		indexPool:        make([]IndexBlock, 0),
		headerBlockPool:  make([]HeaderBlock, 0),
		metadata:         md,
		regionCodeMap:    rm,
		provinceCodeMap:  pm,
		ispCodeMap:       im,
		regionRecordMap:  make(map[string]IndexBlock),
	}

	return mk
}

func (mk *Maker) UseQQWryMake(qFilePath, regionFilePath string) error {
	log.Println("start use qqwry db make ip db")
	qw := NewQQwry(qFilePath)

	provinceMap, err := ProvinceMap(regionFilePath)
	if err != nil {
		return err
	}

	mdData, err := qw.GetQQWryIpRecord(provinceMap)
	if err != nil {
		return err
	}
	mk.metadata = mdData

	m := Metadata{
		StartIP: "255.0.0.0",
		EndIP:   "255.255.255.255",
	}
	m.Format()

	mk.metadata = append(mk.metadata, m)
	return mk.make()
}

func (mk *Maker) make() error {
	var err error
	mk.dbFile, err = os.OpenFile(mk.dbFilePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer func() {
		if err := mk.dbFile.Close(); err != nil {
			log.Fatalf("failed to close db file,err = %s", err)
		}
	}()

	if err := mk.initDBFile(); err != nil {
		return err
	}

	log.Println("+-Try to write the data blocks")
	for _, n := range mk.metadata {
		ib, err := mk.addDataBlock(n)
		if err != nil {
			return err
		}
		mk.indexPool = append(mk.indexPool, *ib)
	}

	log.Println("|--[Ok]")
	log.Println("+-Try to write index blocks ... ")

	startIpN := mk.indexPool[0].startIP
	indexStartPrt, err := mk.dbFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	log.Printf("+- index start pointer %d \n", indexStartPrt)
	mk.headerBlockPool = append(mk.headerBlockPool, HeaderBlock{
		indexStartIp: startIpN,
		indexPtr:     int(indexStartPrt),
	})

	blockLength := mk.indexBlockLength
	counter, shotCounter := 0, mk.indexBlockSize/blockLength-1

	for _, n := range mk.indexPool {
		counter++
		if counter >= shotCounter {
			ptr, err := mk.dbFile.Seek(0, io.SeekCurrent)
			if err != nil {
				return err
			}
			mk.headerBlockPool = append(mk.headerBlockPool, HeaderBlock{
				indexStartIp: n.startIP,
				indexPtr:     int(ptr),
			})
			counter = 0
		}
		_, err := mk.dbFile.Write(n.getBytes())
		if err != nil {
			return err
		}
	}

	if counter > 0 {
		ib := mk.indexPool[len(mk.indexPool)-1]
		ptr, err := mk.dbFile.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		mk.headerBlockPool = append(mk.headerBlockPool, HeaderBlock{
			indexStartIp: ib.startIP,
			indexPtr:     int(ptr),
		})

	}

	log.Println("|--[Ok]")

	log.Println("+-Try to write the super blocks ... ")
	indexEndPrt, err := mk.dbFile.Seek(0, io.SeekCurrent)
	indexEndPrt -= int64(blockLength)
	log.Printf("+- index end pointer %d \n", indexEndPrt)

	if err != nil {
		return err
	}

	_, err = mk.dbFile.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	bs := make([]byte, 8)
	WriteIntLong(bs, 0, indexStartPrt)
	WriteIntLong(bs, 4, indexEndPrt)
	if _, err := mk.dbFile.Write(bs); err != nil {
		return err
	}

	log.Println("|--[Ok]")

	log.Println("+-Try to write the header blocks ... ")
	for _, n := range mk.headerBlockPool {

		if _, err := mk.dbFile.Write(n.Bytes()); err != nil {
			return err
		}
	}
	_, err = mk.dbFile.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	log.Println("|--[Ok]")
	mk.dbFile.Write([]byte("Created by PPIO at " + time.Now().String()))
	log.Println("make db finish")
	return nil
}

func (mk *Maker) initDBFile() error {
	if _, err := mk.dbFile.Seek(0, 0); err != nil {
		return err
	}
	if _, err := mk.dbFile.Write(make([]byte, 8)); err != nil {
		return err
	}
	if _, err := mk.dbFile.Write(make([]byte, mk.totalHeaderSize)); err != nil {
		return err
	}
	log.Println("+-Db file initialized.")
	return nil
}

func (mk *Maker) addDataBlock(md Metadata) (*IndexBlock, error) {

	sIpN, err := IpString2Int64(md.StartIP)
	//fmt.Printf("ip %20s , %-d \n",md.StartIP,sIpN)
	if err != nil {
		return nil, err
	}
	eIpN, err := IpString2Int64(md.EndIP)
	if err != nil {
		return nil, err
	}

	if ib, ok := mk.regionRecordMap[md.RegionString()]; ok {

		ib.startIP = sIpN
		ib.endIP = eIpN

		return &ib, nil
	}

	prt, err := mk.dbFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	dataBlock := md.toDateBlock()
	dataBlock.regionId = mk.regionCodeMap[dataBlock.province]
	dataBlock.provinceId = mk.provinceCodeMap[dataBlock.province]
	dataBlock.ispId = mk.ispCodeMap[dataBlock.isp]

	dataBytes := dataBlock.Bytes()

	dataLen, err := mk.dbFile.Write(dataBytes)
	if err != nil {
		return nil, err
	}

	ib := IndexBlock{
		startIP: sIpN,
		endIP:   eIpN,
		dataPtr: int(prt),
		dataLen: dataLen,
	}
	mk.regionRecordMap[md.RegionString()] = ib

	return &ib, err
}
