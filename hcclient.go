package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"path"
	"time"
)

var listenFlag = flag.Int("l", DEFAULT_COMM_PORT, "listen on port, only for monitored")
var dirFlag = flag.String("d", ".", "received files directory, only for monitored")

func clientmain() {
	fmt.Println("Starting HC monitored-side...")

	// 创建 listener
	listenonport := fmt.Sprintf(":%d", *listenFlag)
	listener, err := net.Listen("tcp", listenonport)
	if err != nil {
		fmt.Println("Error listening", err.Error())
		return //终止程序
	}

	fmt.Println("Files saving directory is", *dirFlag)

	// 监听并接受来自客户端的连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting", err.Error())
			return // 终止程序
		}
		fmt.Printf("Accept connect: %v\n", conn.RemoteAddr())
		go doServerStuff(conn)
	}
}

func doServerStuff(conn net.Conn) {

	var tmphead commhead
	head := make([]byte, binary.Size(tmphead))

	var filename string
	var filefullname string
	var filesessfullname string
	var filesize int64
	var ok bool

	defer conn.Close()
	for {
		_, err := conn.Read(head)
		if err != nil {
			fmt.Println("Connection Closed")
			return //终止程序
		}

		cmmhead := NewCommHead(head)

		switch cmmhead.Command {
		case Comm_File_Name:
			filenamebuf := make([]byte, cmmhead.Datalen)
			rlen, _ := conn.Read(filenamebuf)
			filename = string(filenamebuf[:rlen])
			fmt.Printf("%v ", filename)

			filefullname = path.Join(*dirFlag, filename)
			filesessfullname = filefullname + SESSNAME

			//reply

			hd := new(commhead)
			if ok, filesize = FileExist(filesessfullname); ok {
				//will receive the data from offset that is filesize
				_, err = hd.sendHead(conn, Comm_OK, filesize, 10)
			} else {
				_, err = hd.sendHead(conn, Comm_OK, 0, 10)
			}
			if err != nil {
				fmt.Printf("Send Reply Head failed: %v\n", err)
				return
			}

		case Comm_File_Data:

			filedatabuf := make([]byte, 4096)

			fp, err := os.OpenFile(filesessfullname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Println("cannot OpenFile", filesessfullname, err)
				return
			}
			fp.Seek(0, os.SEEK_END)

			defer fp.Close()

			var receivedlen int64 = filesize
			var receivedlenleft int64 = cmmhead.Datalen - filesize
			progress := 0

			var ll int
			if filesize > 0 {
				fmt.Printf("...")
			} else {
				fmt.Printf("[")
			}

			for {
				if receivedlen == cmmhead.Datalen {
					fmt.Println("]")
					break
				}

				err = conn.SetReadDeadline(time.Now().Add(time.Second * 5))
				if err != nil {
					fmt.Printf("SetReadDeadline failed: %v\n", err)
					return
				}

				if receivedlenleft >= 4096 {
					ll, err = conn.Read(filedatabuf)
				} else {
					ll, err = conn.Read(filedatabuf[:receivedlenleft])
				}

				if err != nil {
					fmt.Printf("conn.Read %v recv: %d ", err, receivedlen)
					return
				}

				_, err = fp.Write(filedatabuf[:ll])
				if err != nil {
					fmt.Println("fp.Write", err)
					return
				}
				receivedlen += (int64)(ll)
				receivedlenleft -= (int64)(ll)
				per := (int)(receivedlen * 100 / cmmhead.Datalen)
				if per > progress && per%2 == 0 {
					fmt.Print("#")
					progress = per
				}
			}

			err = os.Rename(filesessfullname, filefullname)
			if err != nil {
				fmt.Println("os.Rename", filefullname, err)
				return
			}

		case Comm_File_Time:

			databuf := make([]byte, cmmhead.Datalen)
			err = conn.SetReadDeadline(time.Now().Add(time.Second * 5))
			if err != nil {
				fmt.Printf("SetReadDeadline failed: %v\n", err)
				return
			}

			_, err = conn.Read(databuf)
			if err != nil {
				fmt.Println("conn.Read", err)
				return
			}

			tm, err := time.Parse(TMFMT, string(databuf))
			if err == nil {
				os.Chtimes(filefullname, tm, tm)
			}
		case Comm_File_MOD:
			fileInfo, err := os.Stat(filefullname)
			if err != nil {
				fmt.Printf("File: %v os.Stat %v, cannot send\n", filefullname, err)
				return
			}
			mod := fileInfo.Mode()
			mod = (os.FileMode)(cmmhead.Datalen)
			os.Chmod(filefullname, mod)

			hd := new(commhead)
			_, err = hd.sendHead(conn, Comm_File_END, 0, 5)
			if err != nil {
				fmt.Printf("Send Comm_File_END failed: %v\n", err)
				return
			}
		}
	}
}
