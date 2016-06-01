package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"golang.org/x/exp/inotify"
	//	"github.com/howeyc/fsnotify"
)

//*********structure to get json data************
type File struct {
	Dir      string `json:"dir"`
	Onchange string `json:"onchange"`
}

func (f File) toString() (string, string) {
	return toJson(f)
}

//***********to decode json data*******
func toJson(f File) (string, string) {
	_, err := json.Marshal(f)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return f.Dir, f.Onchange
}

//*************function to read json file  (contains direcories tobe monitor and corresponnding action)************
func getFiles() []File {
	raw, err := ioutil.ReadFile("./files.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var c []File
	json.Unmarshal(raw, &c)
	return c
}

//**********MAIN*************

func main() {

	files := getFiles()

	monitor(files)

}

//***********function to get files in maps and call watcher********
func monitor(files []File) {

	var lists []map[string]string

	for _, f := range files {
		l := make(map[string]string)
		m, n := f.toString()
		l["dir"] = m
		l["onchange"] = n
		lists = append(lists, l)
		// fmt.Println(m, "->", n)
		// // fmt.Println(lists)
	}

	for _, k := range lists {
		go watch(k)
	}

	jobj := make(map[string]string)
	jobj["dir"] = "/home/anil/go/src/muto_work/notify/develop/files.json"
	jobj["onchange"] = "/home/anil/go/src/muto_work/notify/develop/test/anil/json.sh"
	watch(jobj)

}

//**********function to implement watcher(inotify)************
func watch(current map[string]string) {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("watching: ", current["dir"])
	done := make(chan bool)
	err = watcher.Watch(current["dir"])
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				//fmt.Println("file has changed changed")
				log.Println("event:", ev)
				out, err := exec.Command(current["onchange"]).Output()
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf(string(out))
			case err := <-watcher.Error:
				fmt.Println("file checking has error")
				log.Println("error:", err)
			}
		}
	}()

	<-done
	watcher.Close()
}
