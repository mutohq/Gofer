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
	workingdir, err = filepath.Abs(filepath.Dir(os.Args[0])) //getting file path from run time
	fmt.Println(workingdir)
	if err != nil {
		log.Fatal(err)
	}

	files, selfobserve := getFiles(workingdir + "/files.json")

	globallock.Lock()
	reqmap = observercreator(files)
	globallock.Unlock()

	startOberver(selfobserve)
}

var reqmap map[string]observedfile
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

		eventlist := make(map[string]int64) //to initialise Blank eventlist
		newfile := observedfile{eventlist: eventlist, onchange: n}
		temp[m] = newfile

	}
	return temp
}

func startOberver(selfobserve bool) {
	if selfobserve {
		//**************to monitor .json file
		jsonobj := observedfile{onchange: workingdir + "/actions/json.sh"}
		for key, value := range reqmap {
			go value.watch(key)
		}

		globallock.Lock()
		reqmap[workingdir+"/files.json"] = jsonobj
		globallock.Unlock()

		jsonobj.watch(workingdir + "/files.json")

	} else { //if not to monitor json file
		i := 0
		for key, value := range reqmap {
			if i == len(reqmap) {
				value.watch(key)
			}
			i++
			go value.watch(key)
		}
	}
}

//***********structure used for creating attributes to monitor file********
type observedfile struct {
	eventlist map[string]int64 // map to store events("MODIFY","DELETE") and their timestamps
	onchange  string           //map to store workingdir PATH and exec EVENT file path map("workingdir":"exec")
	mutex     sync.Mutex
	mywatcher *fsnotify.Watcher
}

//************functions accessed by only instance of structure i.e. observedfile
func execute(obj string) { // executes events of the events File
	len := len(reqmap[obj].eventlist) //to check either eventlist is empty or not
	if len != 0 {
		if obj == workingdir+"/files.json" {
			destroytillnow()
			main()
		}
		// fmt.Println("\n\neventlist of ", obj.address["workingdir"], " :", obj.eventlist, "at", time.Now().Unix())
		out, err := exec.Command(reqmap[obj].onchange).Output()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\n\nevent_list :", string(out), " : executed on: ", obj, "at", time.Now().Unix())
		delete(reqmap[obj].eventlist, "MODIFY")
		delete(reqmap[obj].eventlist, "DELETE")
		fmt.Println("\n\n", reqmap)
	}

}
func destroytillnow() {
	for key, _ := range reqmap {
		fmt.Println("REMOVING WATCHER on::", key)
		// value.mywatcher.RemoveWatch(key)
	}
	<-time.After(time.Second * 2) //time given to stop all concurrently running watchers
	for key, _ := range reqmap {
		globallock.Lock()
		delete(reqmap, key)
		globallock.Unlock()
	}
	fmt.Println(reqmap)

}

//*************** To synchronize eventlist
func queue(obj string) {
	<-time.After(time.Millisecond * 200) //wait when an event is fired
	execute(obj)
}

//**********function to implement watcher(fsnotify)************
func (current observedfile) watch(obj string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\n\nwatching: ", obj)
	done := make(chan bool)
	err = watcher.Watch(obj)
	if err != nil {
		log.Fatal(err)
	}
	current.mywatcher = watcher
	reqmap[obj] = current
	// fmt.Println(current.mywatcher)
	// fmt.Println(reqmap[obj].mywatcher)

	go func() {
		for {
			select {
			case ev := <-watcher.Event:

				//				fmt.Println("\n****************    ", ev.Name, "        ********************")
				if ev.IsModify() {
					//fmt.Println("\nevent:", ev, " at time:", time.Now())
					current.mutex.Lock()
					current.eventlist["MODIFY"] = time.Now().Unix() //update modification timestamp  in structure eventlist
					current.mutex.Unlock()

					current.mutex.Lock()
					queue(obj)
					current.mutex.Unlock()

				}
				if ev.IsDelete() {
					//fmt.Println("\nevent:", ev, " at time:", time.Now())
					current.mutex.Lock()
					current.eventlist["DELETE"] = time.Now().Unix() //update deletion timestamp  in structure eventlist
					current.mutex.Unlock()

					current.mywatcher.RemoveWatch(obj)

					current.mutex.Lock()
					queue(obj)
					current.mutex.Unlock()

					go current.watch(obj)
					done <- true
				}
			case err := <-watcher.Error:
				fmt.Println("\nfile checking has error")
				log.Println("\nerror:", err)
			}
		}
	}()
	<-done
	fmt.Println("\nwatcher closed on :", obj)
}
