package zlog

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"syscall"
	"time"
	"unsafe"
)

// https://github.com/grandecola/mmap

type Zlogger struct {
	data       []byte
	file       *os.File
	ring       *ZlogRing
	logBasePtr uintptr
}

type ZlogRing struct {
	magic       uint8
	ver         uint8
	head        uint16
	tail        uint16
	slotCount   uint16
	slotSize    uint16
	dummy       uint16
	ringMemSize uint32
	logBase     byte
}

type ZlogRead struct {
	Owner uint32
	Start uint16
	Count uint16
}

type Zlog struct {
	Owner uint32
	Data  [252]byte
}

func (zr *ZlogRead) InitOwnerId() {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	zr.Owner = uint32(r1.Intn(1024))
}

func (zr *ZlogRead) GetBytes() []byte {
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.LittleEndian, zr)

	return buffer.Bytes()
}

func (zr *ZlogRead) SetBytes(b []byte) {
	zr.Owner = binary.LittleEndian.Uint32(b[0:4])
	zr.Start = binary.LittleEndian.Uint16(b[4:6])
	zr.Count = binary.LittleEndian.Uint16(b[6:8])
}

func NewZlogger(fileName string, memOrder int) (*Zlogger, error) {
	f, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s, err:%s", fileName, err)
	}

	pageSize := os.Getpagesize()
	mappedSize := int(pageSize * memOrder)

	fd := int(f.Fd())

	m, err := syscall.Mmap(fd, 0, mappedSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		f.Close()
		return nil, err
	}

	ring := (*ZlogRing)(unsafe.Pointer(&m[0]))

	if mappedSize != int(ring.ringMemSize) {
		return nil, fmt.Errorf("mismatch ring memory size: expected=%u, actual=%u",
			mappedSize, ring.ringMemSize)
	}

	base := uintptr(unsafe.Pointer(&ring.logBase))

	return &Zlogger{
		data:       m,
		file:       f,
		ring:       ring,
		logBasePtr: base,
	}, nil
}

func (z *Zlogger) Close() error {
	if z.data != nil {
		if err := syscall.Munmap(z.data); err != nil {
			_ = err
		}

		z.data = nil
	}

	if z.file != nil {
		if err := z.file.Close(); err != nil {
			_ = err
		}

		z.file = nil
	}

	return nil
}

func (z *Zlogger) Fd() int {
	return int(z.file.Fd())
}

func (z *Zlogger) Index(idx uint16) int {
	idx = idx % z.ring.slotCount

	return int(idx)
}

func (z *Zlogger) GetLog(idx uint16) *Zlog {
	offset := uintptr(int(z.ring.slotSize) * z.Index(idx))
	arr := z.logBasePtr + offset
	zlog := (*Zlog)(unsafe.Pointer(arr))

	return zlog
}

func (z *Zlogger) ReadLog() (*ZlogRead, error) {
	zread := ZlogRead{}
	zread.InitOwnerId()
	b := zread.GetBytes()

	n, err := syscall.Read(z.Fd(), b)
	if err != nil {
		return nil, fmt.Errorf("Read err: %s", err)
	} else {
		_ = n
		zread.SetBytes(b)
	}

	return &zread, nil
}

func (z *Zlogger) DoneReadLog(idx uint16) {
	idx = uint16(z.Index(idx))
	z.ring.tail = idx
}

func (zlog *Zlog) GetMessage() string {
	a := (*[1<<30 - 1]byte)(unsafe.Pointer(&zlog.Data))
	size := bytes.IndexByte(a[:], 0)

	// if you just want a string
	goString := string(a[:size:size])

	return goString

	// if you want a slice pointing to the original memory location without a copy
	// goBytes := a[:size:size]

	//goBytes := make([]byte, size)
	//copy(goBytes, a)
}
