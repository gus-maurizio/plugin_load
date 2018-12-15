package main

import (
		"encoding/json"
		"errors"
		"fmt"
		"github.com/shirou/gopsutil/cpu"
		"github.com/shirou/gopsutil/load"
		log "github.com/sirupsen/logrus"
		"github.com/prometheus/client_golang/prometheus"
		"github.com/prometheus/client_golang/prometheus/promhttp"
		"net/http"
    	"time"
)


var PluginConfig 	map[string]map[string]map[string]interface{}
var PluginData		map[string]interface{}

var NumCpus			int = 1


//	Define the metrics we wish to expose
var loadIndicator = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "sreagent_load_metrics",
		Help: "OS Load Utilization Saturation Errors Throughput Latency",
	}, []string{"use"} )

var loadPercent = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "sreagent_load_average",
		Help: "Host OS Load Average",
	}, []string{"load"} )


func PluginMeasure() ([]byte, []byte, float64) {
	lavg, _ 						:= load.Avg()
	lmsc, _ 						:= load.Misc()
	PluginData["loadaverage"]		= *lavg
	PluginData["loadstats"]			= *lmsc
	// Make it understandable
	// Apply USE methodology for LOAD
	// U: 	Usage (usually throughput/latency indicators)
	//		In this case we define as Load1, Load5, Load15 divided by # of cores
	// S:	Saturation (measured compared to each processor busy)
	// E:	Errors (not applicable for LOAD)
	// Prepare the data
	PluginData["load1m"]		= 100.0 * PluginData["loadaverage"].(load.AvgStat).Load1  / float64(NumCpus)
	PluginData["load5m"]		= 100.0 * PluginData["loadaverage"].(load.AvgStat).Load5  / float64(NumCpus)
	PluginData["load15m"]		= 100.0 * PluginData["loadaverage"].(load.AvgStat).Load15 / float64(NumCpus)
	PluginData["use"]    		= PluginData["load1m"]
	PluginData["load"]    		= PluginData["load1m"]
	PluginData["latency"]  		= 0.00
	PluginData["throughput"]  	= PluginData["load1m"]
	PluginData["throughputmax"] = 100.00
	PluginData["saturation"]    = PluginData["load1m"]
	PluginData["errors"]    	= 0.00

	// Update metrics related to the plugin
	loadIndicator.With(prometheus.Labels{"use":  "load1m"}).Set(PluginData["load1m"].(float64))
	loadIndicator.With(prometheus.Labels{"use":  "load5m"}).Set(PluginData["load5m"].(float64))
	loadIndicator.With(prometheus.Labels{"use":  "load15m"}).Set(PluginData["load15m"].(float64))

	loadIndicator.With(prometheus.Labels{"use":  "utilization"}).Set(PluginData["use"].(float64))
	loadIndicator.With(prometheus.Labels{"use":  "saturation"}).Set(PluginData["saturation"].(float64))
	loadIndicator.With(prometheus.Labels{"use":  "throughput"}).Set(PluginData["throughput"].(float64))
	loadIndicator.With(prometheus.Labels{"use":  "errors"}).Set(PluginData["errors"].(float64))


	myMeasure, _ 	:= json.Marshal(PluginData)
	return myMeasure, []byte(""), float64(time.Now().UnixNano())/1e9
}

func PluginAlert(measure []byte) (string, string, bool, error) {
	// log.WithFields(log.Fields{"MyMeasure": string(MyMeasure[:]), "measure": string(measure[:])}).Info("PluginAlert")
	// var m 			interface{}
	// err := json.Unmarshal(measure, &m)
	// if err != nil { return "unknown", "", true, err }
	alertMsg  := ""
	alertLvl  := ""
	alertFlag := false
	alertErr  := errors.New("")

	// Check that the CPU overall value is within range
	switch {
		case PluginData["load"].(float64) < PluginConfig["alert"]["load"]["low"].(float64):
			alertLvl  = "warn"
			alertMsg  += "Overall LOAD below low design point "
			alertFlag = true
			alertErr  = errors.New("low load")
		case PluginData["load"].(float64) > PluginConfig["alert"]["load"]["engineered"].(float64):
			alertLvl  = "fatal"
			alertMsg  += "Overall LOAD above engineered point "
			alertFlag = true
			alertErr  = errors.New("excessive load")
			// return now, looks bad
			return alertMsg, alertLvl, alertFlag, alertErr
		case PluginData["load"].(float64) > PluginConfig["alert"]["load"]["design"].(float64):
			alertLvl  = "warn"
			alertMsg  += "Overall LOAD above design point "
			alertFlag = true
			alertErr  = errors.New("moderately high load")
	}
	return alertMsg, alertLvl, alertFlag, alertErr
}


func InitPlugin(config string) () {
	if PluginData  		== nil {
		PluginData 		=  make(map[string]interface{},20)
	}
	if PluginConfig  	== nil {
		PluginConfig 	=  make(map[string]map[string]map[string]interface{},20)
	}


	// Register metrics with prometheus
	prometheus.MustRegister(loadIndicator)
	prometheus.MustRegister(loadPercent)
	
	initcpu, _ 		:= cpu.Times(true)
	NumCpus			=  len(initcpu)

	err := json.Unmarshal([]byte(config), &PluginConfig)
	if err != nil {
		log.WithFields(log.Fields{"config": config}).Error("failed to unmarshal config")
	}

	log.WithFields(log.Fields{"pluginconfig": PluginConfig}).Info("InitPlugin")
}


func main() {
	config  := 	`
				{
					"alert": 
					{
						"load":
						{
							"low": 			2,
							"design": 		60.0,
							"engineered":	80.0
						}
					}
				}
				`

	//--------------------------------------------------------------------------//
	// time to start a prometheus metrics server
	// and export any metrics on the /metrics endpoint.
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		http.ListenAndServe(":8999", nil)
	}()
	//--------------------------------------------------------------------------//

	InitPlugin(config)
	log.WithFields(log.Fields{"PluginConfig": PluginConfig}).Info("InitPlugin")
	tickd := 10 * time.Second
	for i := 1; i <= 12; i++ {
		tick := time.Now().UnixNano()
		measure, measureraw, measuretimestamp := PluginMeasure()
		alertmsg, alertlvl, isAlert, err := PluginAlert(measure)
		fmt.Printf("Iteration #%d tick %d \n", i, tick)
		log.WithFields(log.Fields{"timestamp": measuretimestamp, 
					  "measure": string(measure[:]),
					  "measureraw": string(measureraw[:]),
					  "PluginData": PluginData,
					  "alertMsg": alertmsg,
					  "alertLvl": alertlvl,
					  "isAlert":  isAlert,
					  "AlertErr":      err,
		}).Info("Tick")
		time.Sleep(tickd)
	}
}
