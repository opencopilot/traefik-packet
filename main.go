package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/packethost/packngo"
	"github.com/packethost/packngo/metadata"
)

var (
	packetAuth    = os.Getenv("PACKET_AUTH")
	packetProject = os.Getenv("PACKET_PROJ")
	backendTag    = os.Getenv("BACKEND_TAG")
)

func getConfig() (map[string]interface{}, error) {
	config := make(map[string]interface{}, 0)
	if packetAuth == "" {
		return nil, errors.New("no PACKET_AUTH provided")
	}
	packetClient := packngo.NewClientWithAuth("", packetAuth, nil)
	devices, _, err := packetClient.Devices.List(packetProject, &packngo.ListOptions{PerPage: 100})
	if err != nil {
		return nil, err
	}

	backends := make([]packngo.Device, 0)
	for _, device := range devices {
		for _, tag := range device.Tags {
			if tag == backendTag {
				backends = append(backends, device)
				break
			}
		}
	}

	config["frontends"] = map[string]interface{}{
		"web": map[string]interface{}{
			"routes": map[string]interface{}{
				"all": map[string]interface{}{
					"rule": "Path:/",
				},
			},
			"backend": "backend",
		},
	}

	config["backends"] = map[string]interface{}{
		"backend": map[string]interface{}{
			"loadbalancer": map[string]interface{}{
				"method": "drr",
			},
			"servers": map[string]interface{}{},
		},
	}
	for i, backend := range backends {
		var addr string
		for _, a := range backend.Network {
			if a.Management && !a.Public && a.AddressFamily == 4 {
				addr = a.Address
			}
		}
		if addr == "" {
			return nil, errors.New("could not find the private management IP of device")
		}

		backend := config["backends"].(map[string]interface{})["backend"].(map[string]interface{})
		servers := backend["servers"].(map[string]interface{})
		servers["server_"+strconv.Itoa(i)] = map[string]interface{}{
			"weight": 1,
			"url":    "http://" + addr,
		}
	}

	return config, nil
}

func getManagementIPs() (*metadata.AddressInfo, *metadata.AddressInfo, error) {
	device, err := metadata.GetMetadata()
	if err != nil {
		return nil, nil, err
	}
	var pub, priv *metadata.AddressInfo
	for _, addr := range device.Network.Addresses {
		if addr.Management {
			if addr.Public {
				pub = &addr
			} else {
				priv = &addr
			}
		}
	}
	if pub == nil || priv == nil {
		return nil, nil, errors.New("could not find the management IPs")
	}

	return pub, priv, nil
}

func applyConfig() error {
	config, err := getConfig()
	if err != nil {
		return err
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	_, priv, err := getManagementIPs()
	if err != nil {
		return err
	}

	client := &http.Client{}
	url := "http://" + priv.Address.String() + ":8080/api/providers/rest"
	request, err := http.NewRequest("PUT", url, strings.NewReader(string(configJSON)))
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(contents))

	return nil
}

func ensureConfig() {
	for {
		err := applyConfig()
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(10 * time.Second)
	}
}

func main() {
	ensureConfig()
}
