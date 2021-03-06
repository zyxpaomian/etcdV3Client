package etcdv3client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"io/ioutil"
	"strconv"
	"time"
)

var Etcdclient *EtcdClient

type EtcdClient struct {
	Endpoints   []string
	Client      *clientv3.Client
	LeaseID 	clientv3.LeaseID
	Lease   	clientv3.Lease
	DialTimeout int
	ReqTimeout  int
}

func ClientInit(dialTimeout int, reqTimeout int, endpoints []string) error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Duration(dialTimeout) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("create etcd client failed, err: %v", err)
	}
	Etcdclient = &EtcdClient{
		Endpoints:   endpoints,
		Client:      cli,
		DialTimeout: dialTimeout,
		ReqTimeout:  reqTimeout,
	}
	return nil
}

func ClientInitWithCA(etcdCert, etcdCertKey, etcdCa string, dialTimeout int, reqTimeout int, endpoints []string) error {
	cert, err := tls.LoadX509KeyPair(etcdCert, etcdCertKey)
	if err != nil {
		return fmt.Errorf("set Tls Cert Falied, Errorlnfo: %s", err.Error())
	}
	caData, err := ioutil.ReadFile(etcdCa)
	if err != nil {
		return fmt.Errorf("set caData Falied, Errorlnfo: %s", err.Error())
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caData)
	_tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Duration(dialTimeout) * time.Second,
		TLS:         _tlsConfig,
	})

	if err != nil {
		return fmt.Errorf("create etcd client failed, err: %v", err)
	}
	Etcdclient = &EtcdClient{
		Endpoints:   endpoints,
		Client:      cli,
		DialTimeout: dialTimeout,
		ReqTimeout:  reqTimeout,
	}
	return nil
}

func (e *EtcdClient) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.ReqTimeout)*time.Second)
	resp, err := e.Client.Get(ctx, key)
	cancel()
	if err != nil {
		return "", fmt.Errorf("get key: %s failed, err: %v", key, err)
	}
	if len(resp.Kvs) > 1 {
		return "", fmt.Errorf("get key: %s failed, err: have multi values, maybe can use prefix", key)
	}
	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("get key: %s failed, err: have no value, check your key string", key)
	}
	kv := resp.Kvs[0]
	return string(kv.Value), nil
}

func (e *EtcdClient) GetPrefix(key string) (map[string]string, error) {
	resultMap := make(map[string]string)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.ReqTimeout)*time.Second)
	resp, err := e.Client.Get(ctx, key, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return nil, fmt.Errorf("get key: %s failed, err: %v", key, err)
	}
	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("get key: %s failed, err: have no value, check your key string", key)
	}
	for _, kv := range resp.Kvs {
		resultMap[string(kv.Key)] = string(kv.Value)
	}
	return resultMap, nil
}

func (e *EtcdClient) Put(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.ReqTimeout)*time.Second)
	_, err := e.Client.Put(ctx, key, value)
	cancel()
	if err != nil {
		return fmt.Errorf("put key: %s failed, err: Xv", key, err)
	}
	return nil
}

func (e *EtcdClient) Del(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.ReqTimeout)*time.Second)
	_, err := e.Client.Delete(ctx, key)
	cancel()
	if err != nil {
		return fmt.Errorf("del key: %s failed, err: %v", key, err)
	}
	return nil
}

func (e *EtcdClient) DelPrefix(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.ReqTimeout)*time.Second)
	_, err := e.Client.Delete(ctx, key, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return fmt.Errorf("del keyprefix: %s failed, err: %v", key, err)
	}
	return nil
}

func (e *EtcdClient) WatchPrefix(key string, keyChan, valueChan, typeChan chan string) {
	wch := e.Client.Watch(context.Background(), key, clientv3.WithPrefix())
	for item := range wch {
		for _, ev := range item.Events{
			keyChan <- string(ev.Kv.Key)
			valueChan <- string(ev.Kv.Value)
			etype := fmt.Sprintf("%s",ev.Type)
			typeChan <- etype
        	//fmt.Printf("%s %q:%q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
        }
	}
}

func (e *EtcdClient) Lock(key string) error {
	getLock := false
	kv := clientv3.NewKV(e.Client)
	retryTimes := int(300 * 2)
	for i := 0; i < retryTimes; i++ {
		// create lease
		lease := clientv3.NewLease(e.Client)
		// set lease time
		leaseResp, err := lease.Grant(context.TODO(), 300)
		if err != nil {
			return fmt.Errorf("set lease time failed, err: %v", err)
		}
		// get leaseld
		leaseId := leaseResp.ID
		// define txn
		txn := kv.Txn(context.TODO())
		txn.If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
			Then(clientv3.OpPut(key, strconv.FormatInt(int64(leaseId), 10), clientv3.WithLease(leaseId))).
			Else(clientv3.OpGet(key))
		// commit txn
		var txnResp *clientv3.TxnResponse
		if txnResp, err = txn.Commit(); err != nil {
			return fmt.Errorf("set txn resp failed, err: %v", err)
		}
		// return if successed
		if txnResp.Succeeded {
			getLock = true
			break
			// try again
		} else {
			time.Sleep(time.Second * 1)
			continue
		}
	}
	if getLock {
		return nil
	} else {
		return fmt.Errorf("Can not get lock")
	}
}

func (e *EtcdClient) Unlock(key string) error {
	resp, err := e.Get(key)
	if err != nil {
		return err
	}
	leaseld, err := strconv.ParseInt(resp, 10, 64)
	if err != nil {
		fmt.Errorf("conv from string to int failed, err: %v")
	}
	lease := clientv3.NewLease(e.Client)
	lease.Revoke(context.TODO(), clientv3.LeaseID(leaseld))
	return nil
}

func (e *EtcdClient) SetLease(leaseTime int64) error {
    ctx, cancel := context.WithTimeout(context.TODO(), time.Second * 5)
    defer cancel()
	lease := clientv3.NewLease(e.Client)
	leaseResp, err := lease.Grant(ctx, leaseTime)
	if err != nil {
		return fmt.Errorf("set etcd lease failed")
	}
	e.LeaseID = leaseResp.ID
	e.Lease = lease
	return nil
}

func (e *EtcdClient) DataRegister(registerKey, registerData string, respError chan error ) error{
	// var keepResp *clientv3.LeaseKeepAliveResponse
	var keepRespChan <-chan *clientv3.LeaseKeepAliveResponse
	
	_, err := e.Client.Put(context.TODO(), registerKey, registerData, clientv3.WithLease(e.LeaseID))
	if err != nil {
		return fmt.Errorf("save register data failed")
	}
	if keepRespChan, err = e.Lease.KeepAlive(context.TODO(), e.LeaseID); err != nil {
		return fmt.Errorf("set lease failed")
	}

	go func() {
		for {
			select {
			//case keepResp = <-keepRespChan:
			case <- keepRespChan:
				if <- keepRespChan == nil {
					time.Sleep(time.Second * 1)
					respError <- fmt.Errorf("auto release failed")
				}
			}
		}
	}()
	return nil
}