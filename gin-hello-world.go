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
	"net/http"
	"os"
	"strconv"
	"time"
)

const AppName = "example-gin-opensergo"
const Port = 8080

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello world")
	})

	go func() {
		routesInfo := r.Routes()
		processServiceContract(routesInfo, []string{"1.1.1.1"}, make(chan struct{}))
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

	ticker := time.NewTicker(30 * time.Second)
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
