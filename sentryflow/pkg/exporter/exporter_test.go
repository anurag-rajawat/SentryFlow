// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Authors of SentryFlow

package exporter

import (
	"context"
	"fmt"
	"net"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/5GSEC/SentryFlow/protobuf/golang"
	protobuf "github.com/5GSEC/SentryFlow/protobuf/golang"
)

func Test_exporter_GetAPIEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	e := getExporter()

	sfClient, closer := getServer(t, e)
	defer closer()

	stream, err := sfClient.GetAPIEvent(ctx, getClientInfo(t))
	if err != nil {
		t.Fatal(err)
	}

	numOfEvents := 5
	go func() {
		for i := 0; i < numOfEvents; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				populateApiEventChannel(e, 1)
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		default:
			e.putApiEventOnClientsChannel(ctx)
		}
	}()

	t.Run("", func(t *testing.T) {
		count := 0
		for {
			if count == numOfEvents-1 {
				cancel()
			}
			_, err := stream.Recv()
			if err != nil {
				t.Log(err)
				break
			}
			fmt.Println("SUCCESSFUL")
			count++
		}
	})

	wg.Wait()
}

func Test_exporter_SendAPIEvent(t *testing.T) {
	type fields struct {
		UnimplementedSentryFlowServer golang.UnimplementedSentryFlowServer
		apiEvents                     chan *protobuf.APIEvent
		logger                        *zap.SugaredLogger
		clients                       *clientList
	}
	type args struct {
		ctx      context.Context
		apiEvent *protobuf.APIEvent
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *protobuf.APIEvent
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &exporter{
				UnimplementedSentryFlowServer: tt.fields.UnimplementedSentryFlowServer,
				apiEvents:                     tt.fields.apiEvents,
				logger:                        tt.fields.logger,
				clients:                       tt.fields.clients,
			}
			got, err := e.SendAPIEvent(tt.args.ctx, tt.args.apiEvent)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendAPIEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SendAPIEvent() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_exporter_addClientToList(t *testing.T) {
	// Given
	e := exporter{
		clients: &clientList{
			Mutex:  &sync.Mutex{},
			client: make(map[string]chan *protobuf.APIEvent),
		},
	}
	uid := uuid.Must(uuid.NewRandom()).String()

	// When
	want := e.addClientToList(uid)

	// Then
	e.clients.Lock()
	got, exists := e.clients.client[uid]
	e.clients.Unlock()
	if !exists || got != want || got == nil {
		t.Errorf("client not added to the client list correctly")
	}
}

func Test_exporter_deleteClientFromList(t *testing.T) {
	// Given
	e := exporter{
		clients: &clientList{
			Mutex:  &sync.Mutex{},
			client: make(map[string]chan *protobuf.APIEvent),
		},
	}
	uid := uuid.Must(uuid.NewRandom()).String()

	// When
	e.deleteClientFromList(uid, e.addClientToList(uid))

	// Then
	e.clients.Lock()
	got, exists := e.clients.client[uid]
	e.clients.Unlock()
	if exists || got != nil {
		t.Errorf("client not deleted from the client list correctly")
	}
}

func Test_exporter_add_and_delete_client_fromList_concurrently(t *testing.T) {
	e := exporter{
		clients: &clientList{
			Mutex:  &sync.Mutex{},
			client: make(map[string]chan *protobuf.APIEvent),
		},
	}

	numOfClients := 1000
	wg := sync.WaitGroup{}
	wg.Add(numOfClients)

	for i := 0; i < numOfClients; i++ {
		go func() {
			defer wg.Done()

			uid := uuid.Must(uuid.NewRandom()).String()
			connChan := e.addClientToList(uid)

			// Simulate some work
			time.Sleep(time.Duration(rand.IntnRange(1, 100)) * time.Millisecond)

			e.deleteClientFromList(uid, connChan)
		}()
	}

	wg.Wait()

	e.clients.Lock()
	if len(e.clients.client) != 0 {
		t.Errorf("Client list is not empty after concurrent access")
	}
	e.clients.Unlock()
}

func Test_exporter_putApiEventOnClientsChannel(t *testing.T) {
	type fields struct {
		UnimplementedSentryFlowServer golang.UnimplementedSentryFlowServer
		apiEvents                     chan *protobuf.APIEvent
		logger                        *zap.SugaredLogger
		clients                       *clientList
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &exporter{
				UnimplementedSentryFlowServer: tt.fields.UnimplementedSentryFlowServer,
				apiEvents:                     tt.fields.apiEvents,
				logger:                        tt.fields.logger,
				clients:                       tt.fields.clients,
			}
			e.putApiEventOnClientsChannel(tt.args.ctx)
		})
	}
}

func populateApiEventChannel(e *exporter, numberOfApiEvent int) {
	for i := 0; i < numberOfApiEvent; i++ {
		event := getApiEvent()
		//time.Sleep(5 * time.Second)
		//fmt.Println("ADDRESS", &event)
		e.apiEvents <- event
	}
}

func getApiEvent() *protobuf.APIEvent {
	return &protobuf.APIEvent{
		Metadata: &protobuf.Metadata{
			ContextId: 1,
			Timestamp: uint64(time.Now().Unix()),
		},
		Source: &protobuf.Workload{
			Name:      "source-workload",
			Namespace: "source-namespace",
			Ip:        "1.1.1.1",
			Port:      int32(rand.IntnRange(1025, 65536)),
		},
		Destination: &protobuf.Workload{
			Name:      "destination-workload",
			Namespace: "destination-namespace",
			Ip:        "93.184.215.14",
			Port:      int32(rand.IntnRange(80, 65536)),
		},
		Request: &protobuf.Request{
			Headers: map[string]string{
				":authority": "example.com",
				":method":    "GET",
				":path":      "/",
				":scheme":    "http",
			},
			Body: "request body",
		},
		Response: &protobuf.Response{
			Headers: map[string]string{
				":status": "200",
			},
			Body: "response body",
		},
		Protocol: "HTTP/1.1",
	}
}

func getExporter() *exporter {
	return &exporter{
		apiEvents: make(chan *protobuf.APIEvent, 1000),
		logger:    zap.S(),
		clients: &clientList{
			Mutex:  &sync.Mutex{},
			client: make(map[string]chan *protobuf.APIEvent),
		},
	}
}

func getClientInfo(t *testing.T) *protobuf.ClientInfo {
	hostname, err := os.Hostname()
	if err != nil {
		t.Errorf("failed to get hostname: %v", err)
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		t.Errorf("failed to get IP address: %v", err)
	}
	var ip string
	if len(ips) > 0 {
		ip = ips[0].String()
	}

	clientInfo := &protobuf.ClientInfo{
		HostName:  hostname,
		IPAddress: ip,
	}

	return clientInfo
}

func getServer(t *testing.T, e *exporter) (protobuf.SentryFlowClient, func()) {
	listener := bufconn.Listen(101024 * 1024)
	baseServer := grpc.NewServer()
	protobuf.RegisterSentryFlowServer(baseServer, e)
	go func() {
		if err := baseServer.Serve(listener); err != nil {
			t.Errorf("failed to start exporter server: %v", err)
			return
		}
	}()

	resolver.SetDefaultScheme("passthrough")
	conn, err := grpc.NewClient("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Errorf("failed to dial to server: %v", err)
		return nil, nil
	}

	closer := func() {
		if err := listener.Close(); err != nil {
			t.Errorf("failed to close listener: %v", err)
		}
		baseServer.Stop()
	}

	return protobuf.NewSentryFlowClient(conn), closer
}
