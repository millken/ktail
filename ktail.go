package main

import (
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"os"
	"time"
	"strings"
	"sync"
	"runtime"
	"flag"
)
var (
	logDir,listenAddr string
	domainCh = make(chan string)
	Message = websocket.Message
	Clients = make(map[string]*ClientConn)
	DomainMap = make(map[string]int)
)

type ClientConn struct {
	websocket *websocket.Conn
	domain string
}

func worker(ch chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for _c := range ch {
		_, ok := DomainMap[_c]
		if !ok {
			DomainMap[_c] = 1
			go tailWorker(_c)
		}
		log.Printf("domain : %s has %d clients\n", _c, DomainMap[_c])
	}

}

func tailWorker(domain string) {
	var osize, nsize int64
	fname := logDir + domain + "/" + domain + "__.log"
	buf := make([]byte, 1)
	
	log.Printf("domain [%s] start , file : %s\n", domain, fname)
	if file, err := os.Open(fname); err == nil {
		defer file.Close()

		if fi, err := file.Stat(); err == nil {
			osize = fi.Size()
T1:			
			for {
				row := make([]string, 0)
				finfo, err := os.Stat(fname)
				if err != nil {
					log.Printf("finfo err = %q", err)
				}
				nsize = finfo.Size()
				if osize > nsize {
					break;
				}
				//log.Printf("osize=%d, nsize=%d  finfo.size=%d", osize, nsize,  finfo.Size())
				if nsize != osize {
					file.Seek(osize, 0)
					for {
						n, _ := file.Read(buf)
						if n == 0 || string(buf) == "\n" {
							break
						}
						row = append(row, string(buf))
						
					}
					_nums := 0
					for _, client := range Clients {
						if client.domain == domain {
							_nums ++
							Message.Send(client.websocket, strings.Join(row, ""))
						}
					}
					if _nums == 0 {
						break T1
					}
					DomainMap[domain] = _nums + 1
					osize = nsize
				}
				
				time.Sleep(250 * time.Millisecond)

			}
		}
	}
	log.Printf("domain [%s] exit\n", domain)
	delete(DomainMap, domain)
}


func tailServer(ws *websocket.Conn) {
	defer func() {
		if err := ws.Close(); err != nil {
			log.Println("Websocket could not be closed", err.Error())
		}else{
			log.Println("Websocket closed")
		}
	}()
	q := ws.Request().URL.Query()
	domain := q.Get("domain")
	domainCh <- domain
		
	client := ws.Request().RemoteAddr
	c := &ClientConn{ws, domain}
	Clients[client] = c
	for {
		var msg string
		err := Message.Receive(ws, &msg)
		if err != nil {
			delete(Clients, client)
			log.Printf("Websocket receive %s, cleint : %s close\n", err.Error(), client)
			break
		}
	}	
}

func startServer() {
	http.Handle("/", websocket.Handler(tailServer))
	err := http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}	
}

func main() {
	flag.StringVar(&logDir, "dir", "/var/log/", "dir of log path")
	flag.StringVar(&listenAddr, "a", ":1578", `address to listen on; ":1578" (default websocket port)"`)
	flag.Parse()
	numCpus := runtime.NumCPU()
	runtime.GOMAXPROCS(numCpus)
	wg := new(sync.WaitGroup)
	for i := 0; i < numCpus; i++ {
		wg.Add(1)
		go worker(domainCh, wg)
	}
	
	startServer()
	close(domainCh)

	wg.Wait()
}
