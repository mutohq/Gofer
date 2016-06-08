package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/howeyc/fsnotify"
)

//*********structure to get json data************

type File struct {
	Dir  string `json:"dir"`
	Exec string `json:"exec"`
	Path string `json:"path"`
}

type jsons struct {
	Loc         []File `json:"loc"`
	Selfobserve bool   `json:"selfobserve"`
}

type config struct { //structure to read config.json file
	Source   string `json:"source"`
	Log_file string `json: "log_file"`
}

func (p config) configtodata() (string, string) { //function to read content from config.json file
	_, err := json.Marshal(p)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return p.Source, p.Log_file
}

// //*************function to read particular json file  (contains direcories to be monitor and corresponnding action)************
func getFiles(address string) ([]File, bool) {
	raw, err := ioutil.ReadFile(address)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var c []File
	var jsons_file jsons
	json.Unmarshal(raw, &jsons_file)

	c = jsons_file.Loc
	return c, jsons_file.Selfobserve
}

//***********to decode json data*******
func (f File) getStrings() (string, string, string) {
	_, err := json.Marshal(f)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return f.Dir, f.Exec, f.Path
}

//**********MAIN*************

func main() {
	var err error
	mainchan := make(chan bool)

	workingdir, err = filepath.Abs(filepath.Dir(os.Args[0])) //getting file path from run time

	if len(os.Args) > 1 {
		configpath = string(os.Args[1])
	} else {
		configpath = "/home/anil/go/src/Gofer/config.json"
	}
	// fmt.Println(configpath)

	raw, err := ioutil.ReadFile(configpath) //to read config.json file
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var c []config
	json.Unmarshal(raw, &c) //unmarshal []byte into []config

	for _, p := range c {
		src, lg := p.configtodata() //to get individual (source,log) pair from config.json file
		logrecord[src] = lg
	}

	for jsonfile, _ := range logrecord { //read .json file name and add it's files to file list
		recordmapinsertion(jsonfile)
	}

	// for jsonkey, _ := range selfoberverrec {
	startObserver("")
	// }

	//fmt.Println(recordmap)
	// <-time.After(time.Second * 5)
	<-mainchan //to block main function
}

//***********function to insert values(observedfile instance) in recordmap********
func recordmapinsertion(jsonfile string) {
	// fmt.Println("runner called")
	var files []File
	tempfiles, selfobserve := getFiles(jsonfile) //to get filelist and selfobserve from a particular .json file
	for _, v := range tempfiles {
		files = append(files, v)
	}
	// fmt.Println("after get files ,self oberve is ,", selfobserve)
	// fmt.Println(globallock)
	globallock.Lock()
	// selfoberverrec[jsonfile] = selfobserve
	globallock.Unlock()

	globallock.Lock()
	// fmt.Println("acuired globallock")
	for _, f := range files {
		m, n, _ := f.getStrings()
		// if p == "relative" {          //code used for relative paths (relative to working direcory of binary)
		// 	m = workingdir + m
		// 	n = workingdir + n
		// }
		channel := make(chan bool)
		eventlist := make(map[string]int64) //to initialise Blank eventlist
		recordmap[m] = observedfile{eventlist: eventlist, onchange: n, done: channel, parent: jsonfile}
	}
	globallock.Unlock()
	// fmt.Println("globalock released ")

	if selfobserve {
		//**************to monitor .json file
		eventlist := make(map[string]int64)
		channel := make(chan bool)
		jsonobj := observedfile{eventlist: eventlist, onchange: " ", done: channel, parent: jsonfile}

		globallock.Lock()
		// fmt.Println("\nacuired globallock before  json watch in startOberver", globallock)
		// fmt.Println("adding file : ", jsonfile, "   with selfoberve", selfobserve)
		recordmap[jsonfile] = jsonobj
		globallock.Unlock()
	}
}

var configpath string                   //variable for path of config.json file
var logrecord = make(map[string]string) //it maps .json file to corresponding logrecord file
// var selfoberverrec = make(map[string]bool)    //it contains selfoberver corresponding to a particular .json file
var recordmap = make(map[string]observedfile) //map to record all ongoing watchers
var workingdir string                         //to get initial working directory
var globallock = &sync.Mutex{}                //global lock to save global values from concurrency

//******************function to startobserver on recordmap values *************
func startObserver(jsonkey string) {
	if jsonkey != "" { //when again called after modification in a json file
		globallock.Lock()
		// fmt.Println("\nacquired globallock before  go routine in startOberver (selfobserve false )", globallock)
		length := 0
		for _, value := range recordmap {
			if value.parent == jsonkey {
				length++
			}
		}
		globallock.Unlock()

		globallock.Lock()
		// fmt.Println("\nacquired globallock before  go routine in startOberver (selfobserve false )", globallock)
		for key, value := range recordmap {
			if value.parent == jsonkey {
				go watch(key)
			}
		}
		globallock.Unlock()
		// fmt.Println("\nreleased globallock after  go routine in startOberver(selfobserve false)", globallock)

	} else {
		i := 0
		globallock.Lock()
		// fmt.Println("\nacquired globallock before  go routine in startOberver (selfobserve false )", globallock)
		for key, _ := range recordmap {

			if i == len(recordmap) {
				watch(key)
			}

			i++
			go watch(key)
		}
		globallock.Unlock()
		// fmt.Println("\nreleased globallock after  go routine in startOberver(selfobserve false)", globallock)
	}
}

//***********structure used for creating attributes to monitor file********
type observedfile struct {
	eventlist map[string]int64 // map to store events("MODIFY","DELETE") and their timestamps
	onchange  string           //map to store workingdir PATH and exec EVENT file path map("workingdir":"exec")
	// mutex     sync.Mutex        //mutex is lock provided to each instance of structure
	mywatcher *fsnotify.Watcher //watcher for each instance of structure
	done      chan bool
	parent    string
}

//**********function to implement watcher(fsnotify)************
func watch(obj string) {
	// done := make(chan bool)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println("\n\nwatching: ", obj)

	err = watcher.Watch(obj)
	if err != nil {
		fmt.Println(obj)
		log.Fatal(err)
	}
	current := recordmap[obj]
	current.mywatcher = watcher

	globallock.Lock()
	// fmt.Println("\nacquired globallock before  assigning current to", obj, " :", globallock)
	recordmap[obj] = current
	globallock.Unlock()
	// fmt.Println("\nreleased globallock after  assigning current to", obj, " :", globallock)
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if ev.IsModify() {
					// fmt.Println("\nevent:", ev, " at time:", time.Now())

					globallock.Lock()
					// fmt.Println("\nacquired globallock before updating eventlist of ", obj, " :", globallock)
					// fmt.Printf("\n%#v", current)
					current.eventlist["MODIFY"] = time.Now().Unix() //update modification timestamp  in structure eventlist
					globallock.Unlock()

					// fmt.Println("\nreleased globallock after updating eventlist of ", obj, " :", globallock)
					current.process_queue(obj) //execute the eventlist

				}
				if ev.IsDelete() {
					// fmt.Println("\nevent:", ev, " at time:", time.Now())

					globallock.Lock()
					// fmt.Println("\nacquired globallock before updating eventlist of ", obj, " :", globallock)
					// fmt.Printf("%#v", current)
					current.eventlist["DELETE"] = time.Now().Unix() //update deletion timestamp  in structure eventlist
					globallock.Unlock()

					watcher.RemoveWatch(obj) //remove watcher whenever DELETE event occurs

					current.process_queue(obj) //execute the eventlist
					// fmt.Println("\nreleased globallock after updating ev	entlist of ", obj, " :", globallock)

					if obj != current.parent { //if DELETE event is not fired from .json file then only refresh the watcher from here
						go watch(obj)
					}
					current.done <- true
				}
			case err := <-watcher.Error:
				fmt.Println("\nfile checking has error")
				log.Println("\nerror:", err)
			}
		}
	}()
	<-current.done

	// fmt.Println("\nwatcher closed on :", obj)
}

//***********functions to work on key of recordmap so that it can access value(observedfile) pointed by that particular key

//*************** To synchronize eventlist

func (current observedfile) process_queue(obj string) {
	<-time.After(time.Millisecond * 200) //wait when an event is fired
	current.execute(obj)
}
func (current observedfile) execute(obj string) { // executes events of the events File
	globallock.Lock()
	// fmt.Println("PROCESS QUEUE HAS BEEN CALLED")
	len := len(recordmap[obj].eventlist) //to check either eventlist is empty or not
	flag := false
	var out []byte
	if len != 0 {
		parent := current.parent
		if obj == parent {
			current.destroytillnow()
			flag = true
		}

		if !flag {
			var err error
			out, err = exec.Command(recordmap[obj].onchange).Output()
			if err != nil {
				fmt.Println()
				log.Fatal(err)
			}
			// fmt.Println("\n", string(out))
			// fmt.Printf("\n\nevent_list  executed on: ", obj, "at", time.Now().Unix())
			delete(recordmap[obj].eventlist, "MODIFY")
			delete(recordmap[obj].eventlist, "DELETE")
			// fmt.Println("after delet of eventlist")
		}
		// fmt.Println("before updatelog")
		updatelog(obj, parent, string(out)) //function call to update log
	}
	globallock.Unlock()
	if flag {
		recordmapinsertion(obj)
		startObserver(obj)
	}
}

func (current observedfile) destroytillnow() {

	for key, value := range recordmap {
		if value.parent == current.parent {
			// fmt.Println("\n\nREMOVING WATCHER on :", key)
			value.mywatcher.RemoveWatch(key)
			value.done <- true
		}
	}
	//<-time.After(time.Second * 4) //time given to stop all concurrently running watchers

	for key, value := range recordmap { //delete the previous values of recordmap
		if value.parent == current.parent {

			delete(recordmap, key)
		}
	}

}

//************function to update logs corresponding to  .json files
func updatelog(obj string, parent string, out string) {
	// open a file
	f, err := os.OpenFile(logrecord[parent], os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	// fmt.Println(logrecord[parent])
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}

	// don't forget to close it
	defer f.Close()

	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stderr instead of stdout, could also be a file.
	log.SetOutput(f)

	log.WithFields(log.Fields{
		"Event ":  out,
		"on file": obj,
	}).Info("executed")

}
