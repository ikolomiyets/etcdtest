package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc"
	"io/ioutil"
	"time"
)

func main() {
	var etcdCert = "./certs/etcd-client.pem"
	var etcdCertKey = "./ca/etcd-client-key.pem"
	var etcdCa = "./ca/ca.pem"

	cert, err := tls.LoadX509KeyPair(etcdCert, etcdCertKey)
	if err != nil {
		panic(err)
	}

	caData, err := ioutil.ReadFile(etcdCa)
	if err != nil {
		panic(err)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caData)

	_tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}

	start := time.Now()
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"https://localhost:2378"},
		DialTimeout: 5 * time.Second,
		TLS:         _tlsConfig,
	})
	if err != nil {
		panic("cannot connect to the server")
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	resp, err := cli.Get(ctx, "/server/1223")
	cancel()
	if err != nil {
		if err == context.Canceled {
			// grpc balancer calls 'Get' with an inflight client.Close
		} else if err == grpc.ErrClientConnClosing {
			// grpc balancer calls 'Get' after client.Close.
		}
	}

	if resp.Kvs == nil || len(resp.Kvs) == 0 {
		panic("cannot find key /server/1223")
	}

	fmt.Printf("result %s, took %v", resp.Kvs[0].Value, time.Now().Sub(start))
}
