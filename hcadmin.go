package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// 结构体未导出字段不会被json解析,所以要大写
type node struct {
	Name string `json:"Name,omitempty"`
	Ip   string
	Port int
}

func (n node) addr() string {
	return fmt.Sprintf("%s:%d", n.Ip, n.Port)
}

type Nodes struct {
	nodedetails map[string]node
}

type filerecord struct {
	Filename string
	Path     string
}

type SendingFiles struct {
	files map[string]string
}

var sdfiles *SendingFiles
var nodes *Nodes

func adminmain() {

	fmt.Println("Starting HC admin-side...")

	nodestorefile := "nodes.json"
	nodes = NewNodes(nodestorefile)

	sdfile := "sendingfiles.json"
	sdfiles = NewSendingFiles(sdfile)

	inputReader := bufio.NewReader(os.Stdin)

	for {

		fmt.Print("$ ")
		input, _ := inputReader.ReadString('\n')

		trimmedInput := strings.Trim(input, "\r\n")
		comm := strings.Split(trimmedInput, " ")

		switch comm[0] {
		case "q", "Q":
			nodes.save(nodestorefile)
			sdfiles.save(sdfile)
			fmt.Println("Bye")
			return
		case "?", "help":
			usage()
		case "fa": //file add
			sdfiles.add(comm[1:])
		case "fd":
			sdfiles.remove(comm[1:])
		case "fl":
			sdfiles.list()
		case "fs":
			sdfiles.save(sdfile)
		case "na":
			nodes.add(comm[1:])
		case "nd":
			nodes.remove(comm[1:])
		case "ns":
			nodes.save(nodestorefile)
		case "nl":
			nodes.list()
		case "sd":
			var CommandLine = flag.NewFlagSet("", flag.ContinueOnError)
			filesFlag := CommandLine.String("f", "", "files will be sent")
			nodesFlag := CommandLine.String("n", "", "nodes will receive the files")
			err := CommandLine.Parse(comm[1:])
			if err != nil {
				fmt.Println("format error:", err)
			}

			filessnd := strings.Split(*filesFlag, ",")
			nodesrecv := strings.Split(*nodesFlag, ",")
			sndFilesToNodes(filessnd, nodesrecv)
		}
	}

	return
}

func usage() {

	fmt.Println("na : add a node: na name IP port")
	fmt.Println("nl : list all nodes")
	fmt.Println("nd : delete nodes: nd node1 node2")
	fmt.Println("ns : save settings of nodes")
	fmt.Println("fa : add a file: fa filename filepath")
	fmt.Println("fl : list all files")
	fmt.Println("fd : delete files: fd filename1 filename2")
	fmt.Println("fs : save settings of files")
	fmt.Println("sd : send files to nodes, sd -f file1,file2... -n node1,node2...")
	fmt.Println("q/Q : exit the program")
}

////////////////////////////////////////////////////////
//SendingFiles
///////////////////////////////////////////////////////

func NewSendingFiles(filename string) *SendingFiles {
	s := &SendingFiles{
		files: make(map[string]string),
	}

	if err := s.load(filename); err != nil {
		log.Println("Error loading sending files:", err)
	}

	return s
}

func (sf *SendingFiles) load(filename string) error {

	f, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening FILEStore:", err)
		return err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	for err == nil {
		var r filerecord
		if err = d.Decode(&r); err == nil {
			sf.files[r.Filename] = r.Path
		}
	}

	if err == io.EOF {
		return nil
	}
	// error occurred:
	log.Println("Error decoding FILEStore:", err) // map hasn't been read correctly
	return err
}

func (sf *SendingFiles) save(filename string) {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Println("Error opening FILEStore:", err)
		return
	}
	defer f.Close()
	e := json.NewEncoder(f)
	for f_name, f_path := range sf.files {
		r := filerecord{f_name, f_path}
		if err := e.Encode(r); err != nil {
			log.Println("Error saving to FILEStore: ", err)
		}
	}
}

func (sf *SendingFiles) remove(filesname []string) {
	if len(filesname) == 0 {
		fmt.Println("input error, format is: filename1 filename2 ...")
		return
	}

	for _, name := range filesname {
		if _, ok := sf.files[name]; ok {
			delete(sf.files, name)
		} else {
			fmt.Println("File named %s not found", name)
		}
	}
}

func (sf *SendingFiles) add(fileinfo []string) {
	if len(fileinfo) == 2 {
		name := fileinfo[0]
		if _, ok := sf.files[name]; ok {
			fmt.Println("File named %s already existed", name)
			return
		}

		sf.files[name] = fileinfo[1]
	} else {
		fmt.Println("input error, format is: name IP port")
	}
}

func (sf *SendingFiles) list() {
	for name, path := range sf.files {
		fmt.Println(name, path)
	}
}

func (sf *SendingFiles) path(f string) string {
	if p, ok := sf.files[f]; ok {
		return p + f
	}
	return ""
}

////////////////////////////////////////////////////////
//Nodes
///////////////////////////////////////////////////////

func NewNodes(filename string) *Nodes {
	s := &Nodes{
		nodedetails: make(map[string]node),
	}

	if err := s.load(filename); err != nil {
		log.Println("Error loading nodes file:", err)
	}

	return s
}

func (n *Nodes) load(filename string) error {

	f, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening NodeStore:", err)
		return err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	for err == nil {
		var r node
		if err = d.Decode(&r); err == nil {
			n.nodedetails[r.Name] = r
		}
	}

	if err == io.EOF {
		return nil
	}
	// error occurred:
	log.Println("Error decoding NodeStore:", err) // map hasn't been read correctly
	return err
}

func (n *Nodes) save(filename string) {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Println("Error opening NodeStore:", err)
		return
	}
	defer f.Close()
	e := json.NewEncoder(f)
	for _, value := range n.nodedetails {
		if err := e.Encode(value); err != nil {
			log.Println("Error saving to NodeStore: ", err)
		}
	}
}

func (n *Nodes) remove(nodesname []string) {
	if len(nodesname) == 0 {
		fmt.Println("input error, format is: node1 node2 ...")
		return
	}

	for _, name := range nodesname {
		if _, ok := n.nodedetails[name]; ok {
			delete(n.nodedetails, name)
		} else {
			fmt.Println("Node named %s not found", name)
		}
	}
}

func (n *Nodes) list() {
	for _, n := range n.nodedetails {
		fmt.Println(n)
	}
}

func (n *Nodes) add(nodeinfo []string) {
	if len(nodeinfo) >= 2 {
		name := nodeinfo[0]
		if _, ok := n.nodedetails[name]; ok {
			fmt.Println("Node named %s already existed", name)
			return
		}

		var port int
		if len(nodeinfo) == 2 {
			port = DEFAULT_COMM_PORT
		} else {
			var err error
			port, err = strconv.Atoi(nodeinfo[2])
			if err != nil {
				fmt.Println("port input error, format is: name IP port")
				return
			}
		}

		ip := nodeinfo[1]
		a_n := node{name, ip, port}
		n.nodedetails[name] = a_n
	} else {
		fmt.Println("input error, format is: name IP port")
	}
}

func sndFilesToNodes(filesname []string, nodesname []string) {

	if len(filesname) == 0 || len(nodesname) == 0 {
		return
	}

	i := 0
	realsndfiles := make([]string, len(filesname))
	for _, f := range filesname {
		afile, ok := sdfiles.files[f]
		if !ok {
			fmt.Printf("file: %v is unknown, cannot send\n")
		}
		realsndfiles[i] = afile
		i++
	}

	if len(realsndfiles) == 0 {
		return
	}

	for _, n := range nodesname {
		fmt.Printf("sending files to: %v\n", n)

		if anode, ok := nodes.nodedetails[n]; ok {
			conn, err := net.Dial("tcp", anode.addr())
			if err != nil {
				fmt.Printf("node %v net.Dial: %v\n", n, err)
				continue
			}

			for i := 0; i < len(realsndfiles); i++ {
				err = sendFile(realsndfiles[i], conn)
				_, ok = err.(*SendError)
				for ok {
					//send again from last-time position
					conn.Close()
					conn, _ = net.Dial("tcp", anode.addr())
					err = sendFile(realsndfiles[i], conn)
					_, ok = err.(*SendError)
				}
			}
			conn.Close()
		}
	}
}

func sendFile(filefullname string, conn net.Conn) (err error) {

	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(filefullname)
	if err != nil {
		fmt.Printf("File: %v os.Stat %v, cannot send\n", filefullname, err)
		return
	}

	filename := fileInfo.Name()
	fmt.Print(filename)

	hd := new(commhead)

	_, err = hd.send(conn, Comm_File_Name, []byte(filename), 10)
	if err != nil {
		fmt.Println("commhead.send commhead err", err)
		return
	}

	//wait for reply
	err = conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	if err != nil {
		fmt.Printf("SetReadDeadline failed: %v\n", err)
		return
	}

	var tmphead commhead
	head := make([]byte, binary.Size(tmphead))
	_, err = conn.Read(head)
	if err != nil {
		fmt.Println("Error wait reply", err.Error())
		err = &SendError{err}
		return
	}

	replyhead := NewCommHead(head)

	//send file data to peer
	var fp *os.File
	fp, err = os.Open(filefullname)
	if err != nil {
		fmt.Println("cannot open file", filefullname, err)
		return
	}

	fp.Seek(replyhead.Datalen, os.SEEK_SET)

	defer fp.Close()

	_, err = hd.sendHead(conn, Comm_File_Data, fileInfo.Size(), 10)
	if err != nil {
		fmt.Println("commhead.send commhead err", err)
		return
	}

	filesize := fileInfo.Size()
	var sendlen int64 = replyhead.Datalen
	progress := 0

	var readlen int
	var ll int
	buff := make([]byte, 4096)

	if replyhead.Datalen > 0 {
		fmt.Print(" ...")
	} else {
		fmt.Print(" [")
	}

	for {
		readlen, err = fp.Read(buff)
		if err == io.EOF {
			fmt.Println("]")
			break
		}

		if err != nil {
			fmt.Print(err)
			break
		}

		err = conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
		if err != nil {
			fmt.Printf("SetWriteDeadline failed: %v\n", err)
		}

		ll, err = conn.Write(buff[:readlen])
		if err != nil {
			fmt.Println(err)
			err = &SendError{err}
			return
		}

		sendlen += (int64)(ll)
		per := (int)(sendlen * 100 / filesize)
		if per > progress && per%2 == 0 {
			fmt.Print("#")
			progress = per
		}
	}

	//file modified time
	tm := fileInfo.ModTime()

	//UTC
	loc, _ := time.LoadLocation("")
	tm = tm.In(loc)

	modstr := tm.Format(TMFMT)
	//fmt.Println(modstr)
	_, err = hd.send(conn, Comm_File_Time, []byte(modstr), 5)
	if err != nil {
		fmt.Println("commhead.send Comm_File_Time err", err)
		return
	}

	//file mode
	//type FileMode uint32
	mod := fileInfo.Mode()
	_, err = hd.sendHead(conn, Comm_File_MOD, (int64)(mod), 5)
	if err != nil {
		fmt.Println("commhead.send Comm_File_MOD err", err)
		return
	}

	//receive confirm from client
	err = hd.recvHead(conn, 5)

	return
}
