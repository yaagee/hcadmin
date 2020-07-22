//////////////////////////////////
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

//
//communication protocol
//

const DEFAULT_COMM_PORT int = 50000
const TMFMT string = "2006-01-02 15:04:05 +0000 UTC"
const SESSNAME string = ".hcadmin.sess"

type CommType uint32

const (
	Comm_OK        CommType = 0
	Comm_File_Name CommType = 1
	Comm_File_Data CommType = 2
	Comm_File_Time CommType = 3
	Comm_File_MOD  CommType = 4
	Comm_File_END  CommType = 5
)

type commhead struct {
	Command CommType
	Datalen int64
}

func (ch *commhead) sendHead(conn net.Conn, t CommType, len int64, tmout int) (n int, err error) {
	ch.Command = t
	ch.Datalen = len

	defer func() {
		if err != nil {
			err = &SendError{err}
		}
	}()

	n = 0
	err = conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(tmout)))
	if err != nil {
		fmt.Printf("SetWriteDeadline failed: %v\n", err)
		return
	}

	n, err = conn.Write(ch.toStream())
	return
}

func (ch *commhead) send(conn net.Conn, t CommType, data []byte, tmout int) (n int, err error) {
	ch.Command = t
	ch.Datalen = (int64)(len(data))

	defer func() {
		if err != nil {
			err = &SendError{err}
		}
	}()

	n = 0
	err = conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(tmout)))
	if err != nil {
		fmt.Printf("SetWriteDeadline failed: %v\n", err)
		return
	}

	n, err = conn.Write(ch.toStream())
	if err != nil {
		return
	}
	n, err = conn.Write(data)
	return
}

func (ch *commhead) recvHead(conn net.Conn, tmout int) (err error) {

	defer func() {
		if err != nil {
			err = &SendError{err}
		}
	}()

	err = conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(tmout)))
	if err != nil {
		fmt.Printf("SetReadDeadline failed: %v\n", err)
		return
	}

	var tmphead commhead
	head := make([]byte, binary.Size(tmphead))
	_, err = conn.Read(head)

	return
}

func (ch *commhead) toStream() []byte {

	buff := new(bytes.Buffer)

	if err := binary.Write(buff, binary.LittleEndian, ch); err != nil {
		log.Fatal("commhead binary write error:", err)
	}

	//binary.BigEndian.PutUint32(req, (uint32)(ch.command))
	//binary.BigEndian.PutUint32(req, ch.datalen)
	return buff.Bytes()
}

func NewCommHead(buff []byte) *commhead {
	commh := new(commhead)
	//commh.command = (CommType)(binary.BigEndian.Uint32(buff))
	//commh.datalen = binary.BigEndian.Uint32(buff)

	databuff := bytes.NewBuffer(buff)
	if err := binary.Read(databuff, binary.LittleEndian, commh); err != nil {
		log.Fatal("commhead binary read error:", err)
	}
	return commh
}

func FileExist(path string) (exst bool, filesize int64) {

	exst = false
	filesize = 0
	info, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			exst = true
			return
		}
		return
	}

	filesize = info.Size()
	exst = true
	return
}

// A SendError indicates an error in sending data to peer.
type SendError struct {
	Err error // The raw error that precipitated this error, if any.
}

// String returns a human-readable error message.
func (e *SendError) Error() string {
	return fmt.Sprintf("error sending data %v", e.Err)
}
