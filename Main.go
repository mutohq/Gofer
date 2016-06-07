package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/howeyc/fsnotify"
)

//*********structure to get json data************

type File struct {
	Dir  string `json:"dir"`
	Exec string `json:"exec"`
	Path string `json:"path"`
}

type config struct {
	Loc         []File `json:"loc"`
	Selfobserve bool   `json:"selfobserve"`
}

//*************function to read json file  (contains direcories tobe monitor and corresponnding action)************
func getFiles(address string) ([]File, bool) {
	raw, err := ioutil.ReadFile(address)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var c []File
	var config_file config
	json.Unmarshal(raw, &config_file)
	c = config_file.Loc
	return c, config_file.Selfobserve
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
	jsonpath = string(os.Args[1])

	if err != nil {
		log.Fatal(err)
	}
	runner()
	<-mainchan
}

func runner() {
	// fmt.Println("runner called")
	files, selfobserve := getFiles(jsonpath)
	fmt.Println("after get files ,self oberve is ,", selfobserve)
	fmt.Println(globallock)
	globallock.Lock()
	// fmt.Println("acuired globallock")
	recordmap = observercreator(files)
	globallock.Unlock()
	// fmt.Println("globalock released ")

	startOberver(selfobserve)
}

var jsonpath string
var recordmap map[string]observedfile
var workingdir string
var globallock = &sync.Mutex{}

//***********function to get files in maps and call watcher********
func observercreator(files []File) map[string]observedfile {
	//array of structures (to store each file address ,exec function,timer and other functions)
	temp := make(map[string]observedfile)
	for _, f := range files {
		m, n, p := f.getStrings()
		if p == "relative" {
			m = workingdir + m
			n = workingdir + n
		}
		channel := make(chan bool)
		eventlist := make(map[string]int64) //to initialise Blank eventlist
		newfile := observedfile{eventlist: eventlist, onchange: n, done: channel}
		temp[m] = newfile

	}
	return temp
}

func startOberver(selfobserve bool) {
	if selfobserve {
		//**************to monitor .json file
		eventlist := make(map[string]int64)
		channel := make(chan bool)
		jsonobj := observedfile{eventlist: eventlist, onchange: workingdir + "/actions/json.sh", done: channel}

		globallock.Lock()
		// fmt.Println("\nacquired globallock before  go routine in startOberver", globallock)
		for key, _ := range recordmap {
			go watch(key)
		}
		globallock.Unlock()
		// fmt.Println("\nreleased globallock after  go routine in startOberver", globallock)

		globallock.Lock()
		// fmt.Println("\nacuired globallock before  json watch in startOberver", globallock)
		recordmap[jsonpath] = jsonobj
		globallock.Unlock()
		// fmt.Println("\nreleased globallock after  json watch in startOberver", globallock)
		watch(jsonpath)

	} else { //if not to monitor json file
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
	eventlist map[string]int64  // map to store events("MODIFY","DELETE") and their timestamps
	onchange  string            //map to store workingdir PATH and exec EVENT file path map("workingdir":"exec")
	mutex     sync.Mutex        //mutex is lock provided to each instance of structure
	mywatcher *fsnotify.Watcher //watcher for each instance of structure
	done      chan bool
}

//**********function to implement watcher(fsnotify)************
func watch(obj string) {
	// done := make(chan bool)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\n\nwatching: ", obj)

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
					fmt.Println("\nevent:", ev, " at time:", time.Now())

					globallock.Lock()
					// fmt.Println("\nacquired globallock before updating eventlist of ", obj, " :", globallock)
					fmt.Printf("\n%#v", current)
					current.eventlist["MODIFY"] = time.Now().Unix() //update modification timestamp  in structure eventlist
					globallock.Unlock()

					// fmt.Println("\nreleased globallock after updating eventlist of ", obj, " :", globallock)
					process_queue(obj) //execute the eventlist

				}
				if ev.IsDelete() {
					fmt.Println("\nevent:", ev, " at time:", time.Now())

					globallock.Lock()
					// fmt.Println("\nacquired globallock before updating eventlist of ", obj, " :", globallock)
					fmt.Printf("%#v", current)
					current.eventlist["DELETE"] = time.Now().Unix() //update deletion timestamp  in structure eventlist
					globallock.Unlock()

					// fmt.Println("\nreleased globallock after updating eventlist of ", obj, " :", globallock)
					watcher.RemoveWatch(obj) //remove watcher whenever DELETE event occurs

					process_queue(obj) //execute the eventlist

					go watch(obj)
					current.done <- true
				}
			case err := <-watcher.Error:
				fmt.Println("\nfile checking has error")
				log.Println("\nerror:", err)
			}
		}
	}()
	<-current.done

	fmt.Println("\nwatcher closed on :", obj)
}

//***********functions to work on key of recordmap so that it can access value(observedfile) pointed by that particular key

//*************** To synchronize eventlist

func process_queue(obj string) {
	<-time.After(time.Millisecond * 200) //wait when an event is fired
	execute(obj)
}
func execute(obj string) { // executes events of the events File
	globallock.Lock()
	fmt.Println("PROCESS QUEUE HAS BEEN CALLED")
	len := len(recordmap[obj].eventlist) //to check either eventlist is empty or not
	flag := false
	if len != 0 {

		if obj == jsonpath {
			destroytillnow()
			flag = true
		}

		if !flag {
			_, err := exec.Command(recordmap[obj].onchange).Output()
			if err != nil {
				fmt.Println()
				log.Fatal(err)
			}

			fmt.Printf("\n\nevent_list  executed on: ", obj, "at", time.Now().Unix())
			delete(recordmap[obj].eventlist, "MODIFY")
			delete(recordmap[obj].eventlist, "DELETE")
		}
	}
	globallock.Unlock()
	if flag {
		runner()
	}
}

func destroytillnow() {

	for key, value := range recordmap {
		fmt.Println("\n\nREMOVING WATCHER on :", key)

		value.mywatcher.RemoveWatch(key)
		value.done <- true

	}
	//<-time.After(time.Second * 4) //time given to stop all concurrently running watchers

	for key, _ := range recordmap { //delete the previous values of recordmap

		delete(recordmap, key)
	}

}
