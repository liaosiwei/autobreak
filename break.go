package main

import (
	"flag"
	"fmt"
	"math"
	"sort"
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
	for i := 0; i < repeat; i++ {
		time.Sleep(1 * time.Second)

		dur, err := ping.Pinger(host, 3)
		if err != nil {
			fmt.Println(err)
			dur = 10000.0 // if ping fails, the ping time sets to 10s
		}
		fmt.Println("ping ", host, ": ", dur)
		data = append(data, dur)
	}
	return
}

type Config struct {
	Nodes  []string
	Repeat int
}

var configFile string

func main() {
	flag.StringVar(&configFile, "config", "config.toml", "the config file location")
	flag.Parse()
	var config Config
	nodeInfo := make(map[string]float64)

	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		fmt.Println(err)
		return
	}
	var wg sync.WaitGroup
	for _, node := range config.Nodes {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			mean := Mean(Ping(url, config.Repeat))
			nodeInfo[url] = mean
		}(node)
	}
	wg.Wait()
	minValue := 1000000.0
	var minNode string
	for k, v := range nodeInfo {
		if v < minValue {
			minValue = v
			minNode = k
		}
	}
	fmt.Println(minNode, " with min ping ", minValue)
}
