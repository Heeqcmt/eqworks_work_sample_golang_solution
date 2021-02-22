package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type counters struct {
	sync.Mutex
	View            int
	Click           int
	ContentSelected string
	TimeSelected    string
}
type contentArray struct {
	sync.Mutex
	cArray []counters
}

type reqToken struct {
	sync.Mutex
	token int
}

var (
	c = counters{}
	//db variable
	db *sql.DB
	//token for rate limiting 10 requests/10seconds
	token = reqToken{
		token: 10,
	}

	//for storing viewing results
	cArray  = contentArray{}
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
	data := contentArray{}
	_ = json.Unmarshal([]byte(file), &data.cArray)

	//read from sqlitedb
	view, click := selectCounter()
	//print record
	if len(data.cArray) > 0 {
		fmt.Fprintf(w, "View: %v \n Click: %v\n", view, click)
		for i := 0; i < len(data.cArray); i++ {
			fmt.Fprintf(w, "content: %v \n date: %v\n", data.cArray[i].ContentSelected, data.cArray[i].TimeSelected)

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

func uploadCounters(view int, click int) error {

	//json file upload containing the array of content selection
	cArray.Lock()
	file, _ := json.Marshal(cArray.cArray)
	_ = ioutil.WriteFile("mockStore.json", file, 0644)
	cArray.Unlock()

	//sqlite upload containing the current counter data
	updateCounter(view, click)

	return nil
}
func createTable() {

	//sql command for creating the table
	createTableSQL := `CREATE TABLE counters (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"view" integer,
		"click" integer)`

	statement, err := db.Prepare(createTableSQL)

	if err != nil {
		log.Fatal(err.Error())
	}

	//excute the command
	statement.Exec()
}

//initialized the table with 0 and 0
func insertCounter() {
	insertStatement := `INSERT INTO counters(view,click) VALUES (?,?)`
	statement, err := db.Prepare(insertStatement)
	if err != nil {
		log.Fatal(err.Error())
	}
	_, err = statement.Exec(0, 0)
	if err != nil {
		log.Fatal(err.Error())
	}
}

//upload only updates never insert
func updateCounter(view int, click int) {
	updateStatement := `UPDATE counters 
	SET view = ?,
	click = ? 
	WHERE id = 1`

	statement, err := db.Prepare(updateStatement)
	//fmt.Printf("update counter %v %v", view, click)
	if err != nil {
		log.Fatal(err.Error())
	}
	_, err = statement.Exec(view, click)
	if err != nil {
		log.Fatal(err.Error())
	}

}

func selectCounter() (int, int) {
	var click int
	var view int
	row, err := db.Query("SELECT view,click FROM counters WHERE id = 1")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer row.Close()
	for row.Next() {
		row.Scan(&view, &click)

	}

	return view, click

}
func main() {

	//sqlite db prep
	//create database file s
	file, err := os.Create("mockStore.db")
	if err != nil {
		log.Fatal(err.Error())
	}
	file.Close()
	//get sqlite instance
	db, _ = sql.Open("sqlite3", "./mockStore.db")
	defer db.Close()
	//create table
	createTable()
	//initializae table
	insertCounter()

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
				uploadCounters(c.View, c.Click)
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
