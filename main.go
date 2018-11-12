package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
)

type Client struct {
	conn       net.Conn
	method     string
	host       string
	address    string
	dataLength int
	data       [1024]byte
}

func (client *Client) parse() (err error) {
	conn := client.conn
	if conn == nil {
		return nil
	}
	n, err := conn.Read((client.data)[:])
	if err != nil {
		log.Println("conn read err:", err)
		return err
	}
	client.dataLength = n
	fmt.Sscanf(string((client.data)[:bytes.IndexByte((client.data)[:], '\n')]), "%s%s", &client.method, &client.host)
	hostPortURL, err := url.Parse(client.host)
	if err != nil {
		log.Println("url parse err:", err)
		return err
	}

	if hostPortURL.Opaque == "443" { //https访问
		client.address = hostPortURL.Scheme + ":443"
	} else { //http访问
		if strings.Index(hostPortURL.Host, ":") == -1 { //host不带端口， 默认80
			client.address = hostPortURL.Host + ":80"
		} else {
			client.address = hostPortURL.Host
		}
	}
	return
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	l, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Println("Listen err:", err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Accept err:", err)
		}
		go handleClientRequest(&Client{conn: conn})
	}
}

func handleClientRequest(client *Client) {
	//获得了请求的host和port，就开始拨号吧
	err := client.parse()
	if err != nil {
		fmt.Println(err)
	}
	server, err := net.Dial("tcp", client.address)
	if err != nil {
		fmt.Println("client Dial remote err:", err)
		return
	}
	defer server.Close()
	if client.method == "CONNECT" {
		fmt.Fprint(client.conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
	} else {
		server.Write((client.data)[:client.dataLength])
	}

	// 基于流的双goroutine拷贝
	//
	go func() {
		_, err = io.Copy(server, client.conn)
		if err != nil {
			log.Println("client to server err:", err)
		}
	}()
	_, err = io.Copy(client.conn, server)
	if err != nil {
		log.Println("server to client err:", err)
	}
}
