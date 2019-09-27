package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

//Info es
type Info struct {
	Number    int
	Direction string
}

//Process es un proceso ejecutandose
type Process struct {
	Name   string
	PID    string
	Status string
	User   string
}

var estructura = Info{Number: 0, Direction: "http://localhost:8000/"}

func indexFunc(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("index.html")
	t.Execute(w, estructura)
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func readAllProc() []Process {
	file, err := os.Open("/proc/")
	if err != nil {
		log.Fatalf("Error abriendo /proc/ %s", err)
	}
	defer file.Close()
	var procs []Process
	list, _ := file.Readdirnames(0) // 0 para leer todo
	for _, name := range list {
		if !isNumeric(name) {
			continue
		}
		p := readOneProc(name)
		procs = append(procs, p)
	}
	return procs
}

func readOneProc(name string) Process {
	procFile, errorOp := os.Open("/proc/" + name + "/status")
	if errorOp != nil {
		fmt.Println("Error abriendo %s %s", name, errorOp)
	}
	process := Process{}
	scanner := bufio.NewScanner(procFile)
	uid := ""
	i := 0
	for scanner.Scan() {
		if i == 3 {
			break
		}
		text := scanner.Text()
		parts := strings.Split(text, ":\t")
		if parts[0] == "Name" {
			process.Name = parts[1]
		} else if parts[0] == "State" {
			process.Status = parts[1]
		} else if parts[0] == "Uid" {
			uid = parts[1]
		}
	}
	defer procFile.Close()
	process.PID = name
	realUID := strings.Split(uid, "\t")
	out, err := exec.Command("id", "-nu", realUID[0]).Output()
	if err != nil {
		fmt.Println(err)
	}
	process.User = strings.Trim(string(out), "\n")
	return process
}

func getProcessInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	p := readAllProc()
	json.NewEncoder(w).Encode(p)
}

func main() {
	http.HandleFunc("/", indexFunc)
	http.HandleFunc("/process", getProcessInfo)
	http.ListenAndServe(":8000", nil)
	readAllProc()
}
