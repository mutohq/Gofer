package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	//"golang.org/x/exp/inotify"
	"time"

	"github.com/howeyc/fsnotify"
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

	//array of structures (to store each file address ,onchange function,timer and other functions)
	var filelist []observedfile

	for _, f := range files {
		jobj := make(map[string]string)
		m, n := f.toString()
		jobj["dir"] = m
		jobj["onchange"] = n
		eventlist := make(map[string]int64) //to initialise Blank eventlist
		newfile := observedfile{eventlist: eventlist, address: jobj, expirytimestamp: time.Now().Unix() + 5}

		filelist = append(filelist, newfile)
		go newfile.watches()

	}

	//**************to monitor .json file
	jobj := make(map[string]string)
	eventlist := make(map[string]int64)
	jobj["dir"] = "/home/anil/go/src/muto_work/notify/develop/files.json"
	jobj["onchange"] = "/home/anil/go/src/muto_work/notify/develop/test/anil/json.sh"
	jsonfile := observedfile{eventlist: eventlist, address: jobj, expirytimestamp: time.Now().Unix() + 5}
	jsonfile.watches()

}

//***********structure used for creating attributes to monitor file********
type observedfile struct {
	eventlist       map[string]int64  // map to store events("MODIFY","DELETE") and their timestamps
	address         map[string]string //map to store DIR PATH and ONCHANGE EVENT file path map("dir":"onchange")
	expirytimestamp int64             //time when the watcher has to check event_list(list) and perform associated actions
	//  ticker *time.Ticker
}

//************functions accessed by only instance of structure i.e. observedfile
func (obj observedfile) watches() { // calls watcher function
	obj.expirytimestamp = time.Now().Unix() + 5
	obj.watch()
}

func (obj observedfile) execute() { // executes events of the events File

	if obj.eventlist["MODIFY"] > obj.eventlist["DELETE"] {

		out, err := exec.Command(obj.address["onchange"]).Output()
		if err != nil {
			log.Fatal(err)
		}
		obj.expirytimestamp = time.Now().Unix() + 5
		fmt.Printf(string(out))
		delete(obj.eventlist, "MODIFY")
		delete(obj.eventlist, "DELETE")
		fmt.Printf("modify event_list executed\n\n")
	} else {
		delete(obj.eventlist, "MODIFY")
		delete(obj.eventlist, "DELETE")
	}

}

func (obj observedfile) expire() { //to check expirytimesamp of process
	// fmt.Println("\nchecking expiry of", obj.address["dir"])
	if obj.expirytimestamp < time.Now().Unix() { //if time.Now().Unix() > expirytimestamp then execute() the eventlist
		fmt.Println("\nExpired timestamp of", obj.address["dir"])
		obj.execute()
	}
}

//**********function to implement watcher(fsnotify)************
func (current observedfile) watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n\nwatching: ", current.address["dir"])
	done := make(chan bool)

	err = watcher.Watch(current.address["dir"])
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			current.expire() //check expirytimestamp of process
			select {
			case ev := <-watcher.Event:
				if ev.IsModify() {
					fmt.Println("event:", ev, " at time:", time.Now())
					current.eventlist["MODIFY"] = time.Now().Unix() //update modification timestamp  in structure eventlist
				}
				if ev.IsDelete() {
					fmt.Println("event:", ev, " at time:", time.Now())
					current.eventlist["DELETE"] = time.Now().Unix() //update deletion timestamp  in structure eventlist
					current.execute()
					done <- true
					current.watches()
				}

			case err := <-watcher.Error:
				fmt.Println("file checking has error")
				log.Println("error:", err)
			}
		}
	}()

	<-done
	watcher.Close()
}
