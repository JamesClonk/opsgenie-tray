package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/getlantern/systray"
	"github.com/getlantern/systray/example/icon"
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	"github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/skratchdot/open-golang/open"
)

var (
	defaultIcon  []byte
	warningIcon  []byte
	criticalIcon []byte
	configFile   string
	config       Config
)

type Config struct {
	APIKey string `json:"api_key"`
}

func init() {
	flags()
	loadConfig()

	defaultIcon = icon.Data
	warningIcon = icon.Data
	criticalIcon = icon.Data
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
	systray.SetTooltip("Lantern")

	mOpsGenie := systray.AddMenuItem("Open OpsGenie", "Open OpsGenie website")
	//mQuit := systray.AddMenuItem("Quit", "Quit OpsGenie-Tray")

	go func() {
		select {
		// case <-mQuit.ClickedCh:
		// 	fmt.Println("Quitting ...")
		// 	systray.Quit()
		case <-mOpsGenie.ClickedCh:
			open.Run("https://app.opsgenie.com/alert")
		}
	}()

	cli := getOpsGenieClient()
	go func() {
		alerts := getAlerts(cli)
		log.Printf("%#v\n", alerts)
	}()
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

func getAlerts(cli *client.OpsGenieAlertV2Client) []alertsv2.Alert {
	req := alertsv2.ListAlertRequest{}
	req.Limit = 25

	resp, err := cli.List(req)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	log.Printf("%#v\n", resp)
	return resp.Alerts
}
