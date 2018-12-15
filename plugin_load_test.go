package main

import (
		"testing"
		"fmt"
		"time"
)

var config string = `
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

func TestInit(t *testing.T) {
	fmt.Printf("Start TestInit\n")
	InitPlugin(config)
	fmt.Printf("Done TestInit %v\n",PluginConfig)
}

func TestMeasure(t *testing.T) {
	fmt.Printf("Start TestMeasure\n")
	if PluginData == nil {
		fmt.Printf("Init PluginData\n")
		PluginData = make(map[string]interface{},20)
	}
	if PluginConfig == nil {
		fmt.Printf("Init PluginConfig\n")
		PluginConfig = make(map[string]map[string]map[string]interface{},20)
	}

	for i := 1; i <= 5; i++ {
		m, mraw, ts := PluginMeasure()
		fmt.Printf("measure %v %v %v\n",string(m),string(mraw),ts)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Printf("Done TestMeasure %v\n",PluginData)
}

func BenchmarkMeasureOnly(b *testing.B) {
    for i := 0; i < b.N; i++ {
		m, mraw, ts := PluginMeasure()
		if i % 15000 == 1 {fmt.Printf("measure %d %v %v %v\n",i,string(m),string(mraw),ts)}
		//time.Sleep(50 * time.Millisecond)
    }
}

func BenchmarkMeasureAlert(b *testing.B) {
    for i := 0; i < b.N; i++ {
		m, mraw, ts := PluginMeasure()
		alertmsg, alertlvl, isAlert, err := PluginAlert(m)
		if i % 15000 == 1 {fmt.Printf("alert %d %v %v %v\n%v %v %v %v\n\n",i,string(m),string(mraw),ts, alertmsg, alertlvl, isAlert, err)}
		//time.Sleep(50 * time.Millisecond)
    }
}
