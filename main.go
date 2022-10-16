package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/haierspi/multi-node-webhook-go/pkg/httpclient"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Command struct {
	Id      string `json:"id"`
	Command string `json:"command"`
}

func (c *Command) run() (string, error) {
	script := "./" + c.Command
	out, err := exec.Command("bash", "-c", script).Output()
	if err != nil {
		log.Printf("Exec command failed: %s\n", err)
		return "", err
	}
	return string(out), nil
}
func (c *Command) Node() *Node {
	return Cfg.node(c.Id)
}

type Hook struct {
	Key      string    `json:"key"`
	Commands []Command `json:"commands"`
}

type Node struct {
	Id   string `json:"id"`
	Host string `json:"host"`
}

func (n *Node) call(id string, key string, post string) (string, error) {
	defer wg.Done()

	posturl := "http://" + n.Host + "/" + key + "?node=1"

	c, err := httpclient.Post(posturl, post)

	if err != nil {
		log.Printf("Hook ["+key+"]["+id+"] run fail:\n%v\n", err)
		return "", err
	} else {
		if c != "" {
			log.Printf("Hook ["+key+"]["+id+"] run ok:\n%v\n", c)
		} else {
			log.Printf("Hook [" + key + "][" + id + "] run ok!")
		}

	}
	return c, nil
}

type Config struct {
	Nodes []*Node `json:"nodes"`
	Hooks []*Hook `json:"hooks"`
}

func (c *Config) node(id string) *Node {
	for _, node := range c.Nodes {
		if node.Id == id {
			return node
			break
		}
	}
	return nil
}

func (c *Config) hook(key string) *Hook {
	for _, hook := range c.Hooks {
		if hook.Key == key {
			return hook
			break
		}
	}
	return nil
}

var Id string
var Host string
var Cfg *Config
var cfgFile string
var allHooks = make(map[string]bool)
var wg sync.WaitGroup

func handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		reqNodeKey := r.URL.Path[1:]
		isReqNode := r.URL.Query().Get("node") != ""
		reqBody, _ := io.ReadAll(r.Body)

		var reqHook *Hook
		if reqHook = Cfg.hook(reqNodeKey); reqHook == nil {
			w.WriteHeader(http.StatusForbidden) // 403
			log.Printf("Hook [" + reqNodeKey + "] does not exist\n")
			return
		}

		for _, command := range reqHook.Commands {
			if command.Id == Id {
				c, err := command.run()
				if err != nil {
					w.Write([]byte("failed\n"))
					return
				} else {
					if c != "" {
						w.Write([]byte("Hook [" + reqNodeKey + "][" + command.Id + "] run ok!\n"))
						log.Printf("Hook ["+reqNodeKey+"]["+command.Id+"] run ok:\n%v\n", c)
					} else {
						w.Write([]byte("Hook [" + reqNodeKey + "][" + command.Id + "] run ok!\n"))
						log.Printf("Hook [" + reqNodeKey + "][" + command.Id + "] run ok!\n")
					}
				}
			} else if !isReqNode {
				wg.Add(1)
				go command.Node().call(command.Id, reqNodeKey, string(reqBody))
			}

		}

		wg.Wait()
		w.Write([]byte("success\n"))
	}
}

func main() {

	err := cfgFlag()
	if err != nil {
		log.Println("startup parameter:", err)
		return
	}
	err = cfgRead()
	if err != nil {
		log.Println("cfgRead:", err)
		return
	}

	s := &http.Server{
		Addr:           Host,
		Handler:        handle(),
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("s.ListenAndServe err: %v", err)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shuting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exiting")

}
func cfgFlag() error {
	flag.StringVar(&Id, "id", "", "服务器启动ID")
	flag.StringVar(&Host, "host", "", "服务器启动Host")
	flag.StringVar(&cfgFile, "config", "", "配置路径 config path")
	c := flag.String("c", "", "配置路径 config path")
	flag.Parse()
	if *c != "" {
		cfgFile = *c
	}
	if Id == "" {
		return errors.New("id parameter must be set, Usage view -help ")
	}
	if cfgFile == "" {
		return errors.New("config parameter must be set, Usage view -help ")
	}
	return nil
}

func cfgRead() error {

	buf, err := os.ReadFile(cfgFile)
	if err != nil {
		return errors.Wrap(err, "Read config file failed:")
	}

	err = json.Unmarshal(buf, &Cfg)
	if err != nil {
		return errors.Wrap(err, "Unmarshal config failed:")
	}

	if Host == "" {
		for _, node := range Cfg.Nodes {

			if node.Id == Id {
				Host = node.Host
				break
			}
		}
		if Host == "" {
			return errors.New("Host Init error")
		}
	}

	return nil

}

// --------------------------------------------------------------------------------
