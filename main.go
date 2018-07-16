package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/JamesClonk/opsgenie-tray/icons"
	"github.com/getlantern/systray"
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	"github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/skratchdot/open-golang/open"
)

var (
	defaultIcon  []byte
	warningIcon  []byte
	criticalIcon []byte
	critical     bool
	warning      bool
	flash        bool
	configFile   string
	config       Config
)

type Config struct {
	APIKey   string `json:"api_key"`
	ShowQuit bool   `json:"show_quit"`
}

func init() {
	flags()
	loadConfig()

	defaultIcon = icons.Base
	warningIcon = icons.Blue
	criticalIcon = icons.Red
}

func flags() {
	flag.StringVar(&configFile, "config", "config.json", "Configuration file")
	flag.Parse()
}

func loadConfig() {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		log.Fatalf("%v\n", err)
	}
}

func main() {
	onExit := func() {
		fmt.Println("Exiting ...")
	}
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(defaultIcon)
	systray.SetTitle("OpsGenie-Tray")
	systray.SetTooltip("OpsGenie Alerts")

	// Menu entry for opening up OpsGenie Alert website
	mOpsGenie := systray.AddMenuItem("Open OpsGenie Alerts", "Open OpsGenie alert website")
	systray.AddSeparator()

	// add Alert entries into array, to be dynamically updated later on
	mAlerts := make([]*systray.MenuItem, 0)
	for i := 0; i <= 9; i++ {
		mAlerts = append(mAlerts, systray.AddMenuItem("Alert", "OpsGenie Alert"))
		mAlerts[i].Hide()
	}

	// optional Quit entry
	var mQuit *systray.MenuItem
	if config.ShowQuit {
		systray.AddSeparator()
		mQuit = systray.AddMenuItem("Quit", "Quit OpsGenie-Tray")
	}

	// API request ticker
	requestTicker := time.NewTicker(20 * time.Second)
	cli := getOpsGenieClient()
	checkAlerts(cli, mAlerts)

	// main event loop
	go func() {
		for {
			select {
			case <-mOpsGenie.ClickedCh:
				open.Run("https://app.opsgenie.com/alert")
			case <-requestTicker.C:
				checkAlerts(cli, mAlerts)
			}
		}
	}()

	// flasher
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			if critical || warning {
				if flash {
					if critical {
						systray.SetIcon(criticalIcon)
					} else if warning {
						systray.SetIcon(warningIcon)
					}
				} else {
					systray.SetIcon(defaultIcon)
				}
				flash = !flash
			}
		}
	}()

	if config.ShowQuit {
		<-mQuit.ClickedCh
		fmt.Println("Quitting ...")
		systray.Quit()
	}
}

func checkAlerts(cli *client.OpsGenieAlertV2Client, mAlerts []*systray.MenuItem) {
	alerts := getAlerts(cli, "status: open")

	// reset state
	critical = false
	warning = false

	for _, alert := range mAlerts {
		alert.Hide()
	}
	for i, alert := range alerts {
		if i > 9 {
			break
		}

		level := "warning"
		for _, tag := range alert.Tags { // get level
			if tag == "critical" {
				critical = true // set critical state if we have any such alerts
				level = "critical"
			}
		}

		// only show them on the list if unacknowledged or critical
		if level == "critical" || !alert.Acknowledged {
			mAlerts[i].SetTitle(alert.Message)
			mAlerts[i].Show()

			if !critical { // dont overwrite critical state flag
				warning = true // only flash warning on unacknowledged alerts
			}
		}
	}

	if !critical && !warning {
		systray.SetIcon(defaultIcon)
	}
}

func getOpsGenieClient() *client.OpsGenieAlertV2Client {
	cli := new(client.OpsGenieClient)
	cli.SetAPIKey(config.APIKey)

	alertCli, err := cli.AlertV2()
	if err != nil {
		log.Fatalf("Could not create alert client: %v\n", err)
	}
	return alertCli
}

func getAlerts(cli *client.OpsGenieAlertV2Client, query string) []alertsv2.Alert {
	req := alertsv2.ListAlertRequest{
		Limit:  25,
		Offset: 0,
		//Query:  "status: open AND acknowledged: false",
		Query: query,
	}

	resp, err := cli.List(req)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	return resp.Alerts
}
