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
	Memory string
}

//ProcessInformation almacena toda la info
type ProcessInformation struct {
	Running  int
	Sleeping int
	Stopped  int
	Zombie   int
	All      int
	Idle     int
	List     []Process
}

//CPUInfo guarda una cosa
type CPUInfo struct {
	Usage float64
}

//MEMInfo es una cosa
type MEMInfo struct {
	Total   float64
	Used    float64
	Percent float64
}

var estructura = Info{Number: 0, Direction: "https://35.225.107.177:8000/"}

func indexFunc(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("index.html")
	t.Execute(w, estructura)
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func readAllProc() ProcessInformation {
	file, err := os.Open("/proc/")
	if err != nil {
		log.Fatalf("Error abriendo /proc/ %s", err)
	}
	defer file.Close()
	var procs []Process
	procInfo := ProcessInformation{All: 0, Running: 0, Sleeping: 0, Stopped: 0, Zombie: 0, Idle: 0}
	list, _ := file.Readdirnames(0) // 0 para leer todo
	for _, name := range list {
		if !isNumeric(name) {
			continue
		}
		p := readOneProc(name)
		procInfo.All++
		if strings.HasPrefix(p.Status, "R") {
			procInfo.Running++
		} else if strings.HasPrefix(p.Status, "S") {
			procInfo.Sleeping++
		} else if strings.HasPrefix(p.Status, "T") {
			procInfo.Stopped++
		} else if strings.HasPrefix(p.Status, "Z") {
			procInfo.Zombie++
		} else if strings.HasPrefix(p.Status, "I") {
			procInfo.Idle++
		}
		procs = append(procs, p)
	}
	procInfo.List = procs
	return procInfo
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
	memOut, memErr := exec.Command("ps", "-p", name, "-o", "%mem").Output()
	if memErr != nil {
		fmt.Println(err)
	}
	process.Memory = strings.TrimLeft(strings.Split(string(memOut), "\n")[1], " ")
	return process
}

func getProcessInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	p := readAllProc()
	json.NewEncoder(w).Encode(p)
}

func getCPU() CPUInfo {
	file, err := os.Open("/proc/stat")
	if err != nil {
		fmt.Println(err)

	}
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	text := scanner.Text()
	text = strings.TrimLeft(text, "cpu  ")
	parts := strings.Split(text, " ")
	sum := 0.0
	usage := 0.0
	for i, part := range parts {
		val, _ := strconv.ParseFloat(part, 32)
		sum += val
		if i == 3 || i == 4 {
			usage += val
		}
	}
	defer file.Close()
	percent := (sum - usage) / sum * 100
	return CPUInfo{Usage: percent}
}

func getCPUInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	p := getCPU()
	json.NewEncoder(w).Encode(p)
}

func getMEM() MEMInfo {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		fmt.Println(err)
	}
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	text := scanner.Text()
	text = strings.TrimLeft(text, "MemTotal:        ")
	text = strings.TrimRight(text, " kB")
	memTotal, _ := strconv.ParseFloat(text, 64)
	scanner.Scan()
	text = scanner.Text()
	text = strings.TrimLeft(text, "MemFree:         ")
	text = strings.TrimRight(text, " kB")
	memFree, _ := strconv.ParseFloat(text, 64)
	defer file.Close()
	memFree /= 1024
	memTotal /= 1024
	percent := (memTotal - memFree) / (memTotal) * 100
	return MEMInfo{Total: memTotal, Used: memTotal - memFree, Percent: percent}
}

func getMEMInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	p := getMEM()
	json.NewEncoder(w).Encode(p)
}

func main() {
	http.HandleFunc("/", indexFunc)
	http.HandleFunc("/process", getProcessInfo)
	http.HandleFunc("/cpu", getCPUInfo)
	http.HandleFunc("/mem", getMEMInfo)
	http.ListenAndServe(":8000", nil)
}
