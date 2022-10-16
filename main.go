package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/haierspi/multi-node-webhook-go/pkg/httpclient"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Command struct {
	Id       string `json:"id"`
	Command  string `json:"command"`
	Display  bool   `json:"display"`
	ParmBind string `json:"parm_bind"`
}

func (c *Command) run(parms []string) (string, error) {
	ext := filepath.Ext(c.Command)
	var script string
	if ext != "" && strings.ToLower(ext) == ".sh" && c.Command[0:1] != "/" {
		script = "./" + c.Command
	} else {
		script = c.Command
	}
	if len(parms) > 0 {
		script += " " + strings.Join(parms, " ")
	}
	cmdd := []string{"bash", "-c", script}
	cmd := exec.Command(cmdd[0], cmdd[1:]...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.Wrap(err, strings.Join(cmdd, " ")+":\n\t"+stderr.String())
	}
	return strings.Join(cmdd, " ") + ":\n" + out.String(), nil
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

func (n *Node) call(command Command, key string, pushChan chan Msg, post []byte) {
	defer wg.Done()

	if command.Id == Id {

		parmBind := make(map[string]string)
		parms := []string{}

		if command.ParmBind != "" {
			json.Unmarshal([]byte(command.ParmBind), &parmBind)
			for n, grep := range parmBind {

				myRegex, _ := regexp.Compile(grep)
				match := myRegex.FindStringSubmatch(string(post))
				if len(match) >= 2 {
					parms = append(parms, n+match[1])
				}
			}
		}
		c, err := command.run(parms)
		if err != nil {
			pushChan <- Msg{
				Err:     err,
				RunOut:  "",
				Name:    "Hook [" + key + "][" + command.Id + "]",
				Command: command,
			}
			return
		} else {
			pushChan <- Msg{
				Err:     nil,
				RunOut:  c,
				Name:    "Hook [" + key + "][" + command.Id + "]",
				Command: command,
			}
		}
	} else {
		posturl := "http://" + n.Host + "/" + key + "?node=1"
		c, err := httpclient.Post(posturl, string(post))

		if err != nil {
			pushChan <- Msg{
				Err:     err,
				RunOut:  "",
				Name:    "Hook [" + key + "][" + command.Id + "]",
				Command: command,
			}
			return
		} else {
			pushChan <- Msg{
				Err:     err,
				RunOut:  c,
				Name:    "Hook [" + key + "][" + command.Id + "]",
				Command: command,
			}
		}
		return
	}
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

type Msg struct {
	Err     error
	RunOut  string
	Name    string
	Command Command
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
		var pushChan = make(chan Msg, 10)

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
				wg.Add(1)
				go command.Node().call(command, reqNodeKey, pushChan, reqBody)
			} else if !isReqNode {
				wg.Add(1)
				go command.Node().call(command, reqNodeKey, pushChan, reqBody)
			}
		}

		go func() {
			wg.Wait()
			close(pushChan)
		}()

		for msg := range pushChan {
			if msg.Err != nil {
				log.Printf(msg.Name+" run fail:\n%v\n", msg.Err)
				if msg.Command.Display && msg.Err != nil {
					w.Write([]byte(fmt.Sprintf(msg.Name+" run fail:\n%v\n", msg.Err)))
				} else {
					w.Write([]byte(msg.Name + " run fail!\n"))
				}

			} else {
				if msg.RunOut != "" {
					log.Printf(msg.Name+" run ok:\n%v\n", msg.RunOut)
				} else {
					log.Printf(msg.Name + " run ok!\n")
				}
				if msg.Command.Display && msg.RunOut != "" {
					w.Write([]byte(fmt.Sprintf(msg.Name+" run ok:\n%v\n", msg.RunOut)))
				} else {
					w.Write([]byte(msg.Name + " run ok!\n"))
				}
			}
		}
		if !isReqNode {
			w.Write([]byte("success\n"))
		}
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
