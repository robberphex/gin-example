package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	service_contract_v1 "github.com/opensergo/opensergo-go/proto/service_contract/v1"
	google_grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

const AppName = "example-gin-opensergo"
const Port = 8080

var ignoreNets = []string{
	"30.39.179.16/30",
	"fe80::1/24",
}
var allowNets = []string{}

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello world")
	})

	go func() {
		routesInfo := r.Routes()
		processServiceContract(routesInfo, getRegIp(), make(chan struct{}))
	}()

	r.Run(":" + strconv.Itoa(Port))
}

func processServiceContract(routesInfo gin.RoutesInfo, ips []string, stopChan chan struct{}) {
	processedName := make(map[string]bool)
	service := service_contract_v1.ServiceDescriptor{}
	for _, routeInfo := range routesInfo {
		method := service_contract_v1.MethodDescriptor{}
		method.HttpPaths = []string{routeInfo.Path}
		method.HttpMethods = append(method.HttpMethods, routeInfo.Method)
		method.Name += routeInfo.Method + " " + routeInfo.Path
		if _, ok := processedName[method.Name]; !ok {
			service.Methods = append(service.Methods, &method)
			processedName[method.Name] = true
		}
	}
	var addrs []*service_contract_v1.SocketAddress
	for _, ip := range ips {
		addrs = append(addrs, &service_contract_v1.SocketAddress{
			Address:   ip,
			PortValue: uint32(Port),
		})
	}

	req := service_contract_v1.ReportMetadataRequest{
		AppName: AppName,
		ServiceMetadata: []*service_contract_v1.ServiceMetadata{
			{
				ListeningAddresses: addrs,
				Protocols:          []string{"http"},
				ServiceContract: &service_contract_v1.ServiceContract{
					Services: []*service_contract_v1.ServiceDescriptor{
						&service,
					},
				},
			},
		},
	}

	ticker := time.NewTicker(5 * time.Minute)
	reportPeriodically(stopChan, ticker, req)
}

func reportPeriodically(stopChan chan struct{}, ticker *time.Ticker, req service_contract_v1.ReportMetadataRequest) {
	go func() {
		for {
			select {
			case <-stopChan:
				fmt.Println("stopping!!!")
				return
			case <-ticker.C:
				ose := getOpenSergoEndpoint()
				if ose != "" {
					timeoutCtx, _ := context.WithTimeout(context.Background(), 10*time.Second)
					conn, err := google_grpc.DialContext(timeoutCtx, ose, google_grpc.WithTransportCredentials(insecure.NewCredentials()))
					if err != nil {
						fmt.Printf("err: %v\n", err)
					}
					mClient := service_contract_v1.NewMetadataServiceClient(conn)
					reply, err := mClient.ReportMetadata(context.Background(), &req)
					fmt.Printf("ReportMetadata: reply: %v, err: %v\n", reply, err)
					_ = reply
				}
			}
		}
	}()
}

type openSergoConfig struct {
	Endpoint string `json:"endpoint"`
}

func getOpenSergoEndpoint() string {
	var err error
	configStr := os.Getenv("OPENSERGO_BOOTSTRAP_CONFIG")
	configBytes := []byte(configStr)
	if configStr == "" {
		configPath := os.Getenv("OPENSERGO_BOOTSTRAP")
		configBytes, err = ioutil.ReadFile(configPath)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
	}
	config := openSergoConfig{}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	return config.Endpoint
}

func getRegIp() []string {
	var ignoreSubnets []*net.IPNet
	for _, ignoreNet := range ignoreNets {
		_, ignoreSubnet, _ := net.ParseCIDR(ignoreNet)
		ignoreSubnets = append(ignoreSubnets, ignoreSubnet)
	}

	var allowSubnets []*net.IPNet
	for _, allowNet := range allowNets {
		_, allowSubnet, _ := net.ParseCIDR(allowNet)
		allowSubnets = append(allowSubnets, allowSubnet)
	}

	var res []string

	ifaces, _ := net.Interfaces()
	// handle err
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
	addrLoop:
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				for _, ignoreSubnet := range ignoreSubnets {
					if ignoreSubnet.Contains(ipNet.IP) {
						continue addrLoop
					}
				}
				if len(allowSubnets) == 0 {
					res = append(res, ipNet.IP.String())
				} else {
					for _, allowSubnet := range allowSubnets {
						if allowSubnet.Contains(ipNet.IP) {
							res = append(res, ipNet.IP.String())
							continue addrLoop
						}
					}
				}
			}
		}
	}
	return res
}
