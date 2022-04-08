package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc"
	"io/ioutil"
	"sync"
	"time"
)

const (
	iterations = 10000
	timeout    = 300 * time.Second
	workers    = 200
)

var wg sync.WaitGroup
var fork sync.WaitGroup

const (
	Read = iota
	Write
)

type Job struct {
	Operation int
	Key       string
	Value     string
}

func main() {
	var etcdCert = "./certs/etcd.pem"
	var etcdCertKey = "./certs/etcd-key.pem"
	var etcdCa = "./certs/ca.pem"

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
		DialTimeout: timeout,
		TLS:         _tlsConfig,
	})
	if err != nil {
		panic("cannot connect to the server")
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	jobsChan := make(chan Job, workers+10)

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker(ctx, i, cli, &jobsChan, &wg)
	}

	fork.Add(iterations * 2)
	go func() {
		for i := 0; i < iterations; i++ {
			go func(i int) {
				jobsChan <- Job{
					Operation: Write,
					Key:       fmt.Sprintf("/user/9000%d", i),
					Value:     fmt.Sprintf("%d", i),
				}
				fork.Done()
			}(i)
		}
	}()

	go func() {
		for i := 0; i < iterations; i++ {
			go func(i int) {
				jobsChan <- Job{
					Operation: Read,
					Key:       fmt.Sprintf("/user/9000%d", i),
				}
				fork.Done()
			}(i)
		}
	}()

	fork.Wait()
	close(jobsChan)
	wg.Wait()
	fmt.Printf("execution time %v\n", time.Now().Sub(start))
}

func worker(ctx context.Context, id int, cli *clientv3.Client, jobChannel *chan Job, wg *sync.WaitGroup) {
	for {
		job, ok := <-*jobChannel
		if !ok {
			wg.Done()
			return
		}

		switch job.Operation {
		case Read:
			resp, err := cli.Get(ctx, job.Key)

			if err != nil {
				if err == context.Canceled {
					// grpc balancer calls 'Get' with an inflight client.Close
				} else if err == grpc.ErrClientConnClosing {
					// grpc balancer calls 'Get' after client.Close.
				}
			}

			if resp.Kvs == nil || len(resp.Kvs) == 0 {
				fmt.Printf("cannot find key '%v'\n", job.Key)
			} else {
				fmt.Printf("result %s\n", resp.Kvs[0].Value)
			}
		case Write:
			_, err := cli.Put(ctx, job.Key, job.Value)

			if err != nil {
				if err == context.Canceled {
					// grpc balancer calls 'Get' with an inflight client.Close
				} else if err == grpc.ErrClientConnClosing {
					// grpc balancer calls 'Get' after client.Close.
				}
			}
		}
	}
}
