package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type counters struct {
	sync.Mutex
	View            int
	Click           int
	ContentSelected string
	TimeSelected    string
}
type countersArray struct {
	sync.Mutex
	cArray []counters
}

type reqToken struct {
	sync.Mutex
	token int
}

var (
	c = counters{}

	//token for rate limiting 10 requests/10seconds
	token = reqToken{
		token: 10,
	}

	//for storing viewing results
	cArray  = countersArray{}
	content = []string{"sports", "entertainment", "business", "education"}
)

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to EQ Works 😎")
}

func viewHandler(w http.ResponseWriter, r *http.Request) {

	t := time.Now()
	c.Lock()
	c.ContentSelected = content[rand.Intn(len(content))]
	c.TimeSelected = t.Format("Mon Jan _2 15:04:05 2006")
	c.View++
	c.Unlock()
	cArray.Lock()
	//append the selection and view data to cArray for later saving
	cArray.cArray = append(cArray.cArray, c)
	cArray.Unlock()

	//sleep a certain time
	err := processRequest(r)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}

	// simulate random click call
	if rand.Intn(100) < 50 {
		processClick(c.ContentSelected)
	}
	fmt.Fprintf(w, "content: %v \n clicks: %v \n view: %v \n %v", c.ContentSelected, c.Click, c.View, c.TimeSelected)

}

func processRequest(r *http.Request) error {
	time.Sleep(time.Duration(rand.Int31n(50)) * time.Millisecond)
	return nil
}

func processClick(data string) error {
	c.Lock()
	c.Click++
	c.Unlock()

	return nil
}

func statsHandler(w http.ResponseWriter, r *http.Request) {

	if !isAllowed() {
		w.WriteHeader(429)
		fmt.Fprintf(w, "request exceeded limit")
		return
	}
	fmt.Fprintf(w, "stats page\n")

	//read from mockstorage
	file, _ := ioutil.ReadFile("mockStore.json")
	data := countersArray{}
	_ = json.Unmarshal([]byte(file), &data.cArray)

	//print record
	if len(data.cArray) > 0 {
		for i := 0; i < len(data.cArray); i++ {
			fmt.Fprintf(w, "content: %v \n clicks: %v \n view: %v \n %v\n", data.cArray[i].ContentSelected, data.cArray[i].Click, data.cArray[i].View, data.cArray[i].TimeSelected)

		}

	} else {
		fmt.Fprintf(w, "No Data yet")
	}

}

func isAllowed() bool {

	if token.token > 0 {
		token.Lock()
		token.token--
		token.Unlock()
		return true
	}
	return false
}

func uploadCounters() error {
	cArray.Lock()
	file, _ := json.Marshal(cArray.cArray)
	_ = ioutil.WriteFile("mockStore.json", file, 0644)
	cArray.Unlock()
	return nil
}

func main() {

	tickerUpload := time.NewTicker(5 * time.Second)
	tickerReq := time.NewTicker(10 * time.Second)
	done := make(chan bool)

	//excute upload every 5 secondss
	go func() {
		for {
			select {
			case <-done:
				return
			case <-tickerUpload.C:
				fmt.Println("upload called")
				uploadCounters()
			}
		}
	}()

	//reload requestToken every 10 secondss
	go func() {
		for {
			select {
			case <-done:
				return
			case <-tickerReq.C:
				fmt.Println("reload token called")
				token.Lock()
				token.token = 10
				token.Unlock()
			}
		}
	}()
	http.HandleFunc("/", welcomeHandler)
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/stats/", statsHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
