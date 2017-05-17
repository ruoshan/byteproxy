package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"
)

const BufSize = 1024 * 16

var flagSize = flag.Int("s", 1, "number of bytes per TCP packet after throttled")
var flagDelay = flag.Int("d", 0, "delay in ms")
var flagListen = flag.String("l", "", "listen port")
var flagUpstream = flag.String("u", "", "upstream port")

const directionUsage = `throttle direction: cs | sc | both | none
	* cs: throttle client to server data path
	* sc: throttle server to client data path
	* both: throttle both direction (default)
	* none: do not throttle`

var flagDirection = flag.String("r", "both", directionUsage)

func Throttle(from io.Reader, to io.Writer, wg *sync.WaitGroup) {
	defer wg.Done()
	buf := make([]byte, BufSize)
	for {
		if readCount, err := from.Read(buf); err == nil {
			for i := 0; i < readCount; i += *flagSize {
				stepEnd := i + *flagSize
				if stepEnd > readCount {
					stepEnd = readCount
				}
				if writeCount, err := to.Write(buf[i:stepEnd]); err != nil || writeCount != stepEnd-i {
					log.Printf("Failed to write: %s", err)
					return
				}
				time.Sleep(time.Duration(*flagDelay) * time.Millisecond)
			}
		} else if err == io.EOF {
			return
		} else {
			log.Printf("Failed to read: %s", err)
			return
		}
	}
}

func Copy(from io.Reader, to io.Writer, wg *sync.WaitGroup) {
	io.Copy(to, from)
	wg.Done()
}

func Proxy(client *net.TCPConn) {
	defer client.Close()
	client.SetNoDelay(true)
	_upstream, err := net.Dial("tcp", *flagUpstream)
	if err != nil {
		log.Printf("Failed to dial upstream: %s\n", err)
		return
	}
	upstream := _upstream.(*net.TCPConn)
	defer upstream.Close()
	upstream.SetNoDelay(true)

	var wg = new(sync.WaitGroup)
	wg.Add(2)

	switch *flagDirection {
	case "cs":
		go Throttle(client, upstream, wg)
		go Copy(upstream, client, wg)
	case "sc":
		go Throttle(upstream, client, wg)
		go Copy(client, upstream, wg)
	case "none":
		go Copy(client, upstream, wg)
		go Copy(upstream, client, wg)
	case "both":
		fallthrough
	default:
		go Throttle(client, upstream, wg)
		go Throttle(upstream, client, wg)
	}
	wg.Wait()
}

func HandleQuit(l net.Listener) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	log.Printf("Bye")
	l.Close()
}

func checkFlags() error {
	if *flagSize < 1 {
		return errors.New("size")
	}
	if *flagDelay < 0 {
		return errors.New("delay")
	}
	switch *flagDirection {
	case "sc", "cs", "both", "none":
		return nil
	default:
		return errors.New("direction")
	}
}

func main() {
	flag.Parse()
	if err := checkFlags(); err != nil {
		log.Fatalf("Invalid option: %s", err)
	}
	log.Printf("Delay: %dms\n", *flagDelay)
	log.Printf("Listen: %s\n", *flagListen)
	log.Printf("Upstream: %s\n", *flagUpstream)
	log.Printf("Throttle Direction: %s\n", *flagDirection)

	_listener, err := net.Listen("tcp", *flagListen)
	if err != nil {
		log.Fatal(err)
	}
	listener := _listener.(*net.TCPListener)

	go HandleQuit(listener)
	for {
		if c, err := listener.Accept(); err == nil {
			log.Println("Accepted")
			go Proxy(c.(*net.TCPConn))
		} else {
			return
		}
	}
}
