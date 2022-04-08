# ETCD Test
To run the program first you have to bootstrap etcd server.
To do so, create SSL CA and server key pair.

First create etcd.ext file with the following content (in certs directory):
```
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = <your_hostname>
DNS.2 = <your_hostalias>
DNS.3 = localhost
IP.1 = <your_ip>
IP.2 = 127.0.0.1
```

Then run this sequence of commands to generate the CA and server certificate and private key (also all in the certs directory).
```
openssl genrsa -out ca-key.pem 2048
openssl req -x509 -new -nodes -key ca-key.pem -subj "/CN=root-ca" -days 10000 -sha256 -out ca.pem
openssl genrsa -out etcd-key.pem 2048
openssl req -subj '/CN=etcd.dev.iktech.io' -new -sha256 -key etcd-key.pem -out etcd.csr -config etcd.ext
openssl x509 -req -in etcd.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial -out etcd.pem -days 7200 -sha256 -extensions v3_req -extfile etcd.ext
```

Then run the following command to start etcd server with Docker.
```
  export NODE1=<your_ip_address>
  docker run -p 2378:2379 -p 2382:2380 -d -v etcd-data:/etcd-data -v $PWD/certs:/etc/ssl/etcd --name etcd quay.io/coreos/etcd:latest /usr/local/bin/etcd \
  --data-dir=/etcd-data --name node1 \
  --cert-file=/etc/ssl/etcd/etcd.pem \
  --key-file=/etc/ssl/etcd/etcd-key.pem \
  --peer-cert-file=/etc/ssl/etcd/etcd.pem \
  --peer-key-file=/etc/ssl/etcd/etcd-key.pem \
  --trusted-ca-file=/etc/ssl/etcd/ca.pem \
  --peer-trusted-ca-file=/etc/ssl/etcd/ca.pem \
  --peer-client-cert-auth \
  --client-cert-auth \
  --initial-advertise-peer-urls https://$NODE1:2382 \
  --listen-peer-urls https://0.0.0.0:2380 \
  --advertise-client-urls https://$NODE1:2378 \
  --listen-client-urls https://0.0.0.0:2379 \
  --initial-cluster node1=https://$NODE1:2382
```

If Docker running in Windows environment run the following (in PowerShell):
```
$NODE1="<your_ip_address>"
docker run -p 2378:2379 -p 2382:2380 -d -v etcd-data:/etcd-data -v ${PWD}/certs:/etc/ssl/etcd --name etcd quay.io/coreos/etcd:latest /usr/local/bin/etcd --data-dir=/etcd-data --name node1 --cert-file=/etc/ssl/etcd/etcd.pem --key-file=/etc/ssl/etcd/etcd-key.pem --peer-cert-file=/etc/ssl/etcd/etcd.pem --peer-key-file=/etc/ssl/etcd/etcd-key.pem --trusted-ca-file=/etc/ssl/etcd/ca.pem --peer-trusted-ca-file=/etc/ssl/etcd/ca.pem --peer-client-cert-auth --client-cert-auth --initial-advertise-peer-urls https://${NODE1}:2382 --listen-peer-urls https://0.0.0.0:2380 --advertise-client-urls https://${NODE1}:2378 --listen-client-urls https://0.0.0.0:2379 --initial-cluster node1=https://${NODE1}:2382
```

Check connection with the following command `etcdctl --cacert=./certs/ca.pem --cert=./certs/etcd.pem --key=./certs/etcd-key.pem --endpoints=https://${NODE1}:2378 member list `.