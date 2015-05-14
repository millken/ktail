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
	nameCh = make(chan string)
	Message = websocket.Message
	Clients = make(map[string]*ClientConn)
	NameMap = make(map[string]int)
)

type ClientConn struct {
	websocket *websocket.Conn
	name string
}

func worker(ch chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for _c := range ch {
		_, ok := NameMap[_c]
		if !ok {
			NameMap[_c] = 1
			go tailWorker(_c)
		}
		log.Printf("name : %s has %d clients\n", _c, NameMap[_c])
	}

}

func tailWorker(name string) {
	var osize, nsize int64
	fname := logDir + name
	buf := make([]byte, 1)
	
	log.Printf("websocket tail file : %s\n", fname)
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
						if client.name == name {
							_nums ++
							Message.Send(client.websocket, strings.Join(row, ""))
						}
					}
					if _nums == 0 {
						break T1
					}
					NameMap[name] = _nums + 1
					osize = nsize
				}
				
				time.Sleep(250 * time.Millisecond)

			}
		}
	}
	log.Printf("name [%s] exit\n", name)
	delete(NameMap, name)
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
	name := q.Get("name")
	nameCh <- name
		
	client := ws.Request().RemoteAddr
	c := &ClientConn{ws, name}
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
		go worker(nameCh, wg)
	}
	
	startServer()
	close(nameCh)

	wg.Wait()
}
