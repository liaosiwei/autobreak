package main

import (
	"flag"
	"log"
	"math"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/liaosiwei/autobreak/ping"
)

// GetByPercentile will get the data from the percentile value of data,
// the parameter percentile ranges from 1 to 100 as integer
func Percentile(data []float64, percentile int) float64 {
	length := len(data)
	if length < 1 || percentile < 1 || percentile > 100 {
		panic("illegal parameter")
	}
	sort.Float64s(data)
	point := int(math.Ceil((float64(percentile)/100.0)*float64(length))) - 1
	return data[point]
}

func Mean(data []float64) float64 {
	sum := 0.0
	for _, value := range data {
		sum += value
	}
	return sum / float64(len(data))
}

func Ping(host string, repeat int) (data []float64) {
	if repeat < 1 {
		panic("ping time cannot less than 1")
	}
	for i := 0; i < repeat; i++ {
		time.Sleep(1 * time.Second)

		dur, err := ping.Pinger(host, 1)
		if err != nil {
			log.Println(err)
			dur = 10000.0 // if ping fails, the ping time sets to 10s
		}
		log.Println("ping ", host, ": ", dur)
		data = append(data, dur)
	}
	return
}

func SetProxy(proxy string) error {
	out, err := exec.Command("networksetup", "-getautoproxyurl", "Wi-Fi").Output()
	if err != nil {
		panic(err)
	}
	pacPair := strings.Split(string(out), "Enable")

	if len(pacPair) != 2 {
		panic(`command networksetup -getautoproxyurl "Wi-Fi" is not correct`)
	}

	pacAddress := pacPair[0]
	start := strings.Index(pacAddress, "http")
	var pac string
	if start != -1 {
		pac = strings.TrimSpace(pacAddress[strings.Index(pacAddress, "http"):])
		if proxy == pac {
			return nil
		}
	}

	out, err = exec.Command("networksetup", "-setautoproxyurl", "Wi-Fi", proxy).Output()
	if err != nil {
		log.Fatal("switching pac from " + pac + " to " + proxy + " failed: " + string(out))
	}
	log.Println("switching pac from " + pac + " to " + proxy)
	return err
}

type Config struct {
	Nodes  map[string]Host
	Repeat int
}

type Host struct {
	Address   string
	PingValue float64
}

var configFile string

func main() {
	flag.StringVar(&configFile, "config", "config.toml", "the config file location")
	flag.Parse()
	var config Config

	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Fatal(err)
	}
	if config.Repeat < 1 {
		panic("ping time cannot less than 1")
	}

	var Do = func() {
		var wg sync.WaitGroup
		for node, _ := range config.Nodes {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				mean := Mean(Ping(url, config.Repeat))
				host := config.Nodes[url]
				host.PingValue = mean
				config.Nodes[url] = host
			}(node)
		}
		wg.Wait()
		minValue := 1000000.0
		var minNode string
		for k, v := range config.Nodes {
			if v.PingValue < minValue {
				minValue = v.PingValue
				minNode = k
			}
		}
		log.Println("choose ", minNode, " with min ping ", minValue)
		if err := SetProxy(config.Nodes[minNode].Address); err != nil {
			log.Fatal(err)
		}
	}
	for {
		go Do()
		select {
		case <-time.After(time.Duration(2) * time.Minute):
		}
	}
}
