package main

import(
    mce "github.com/zyxpaomian/etcdv3client"
    "fmt"    
)


func main() {
    var endPoints  = []string{"192.168.159.133:2379"}
    err := mce.ClientInit(10, 10, 300, endPoints)
    if err != nil {
        fmt.Println("etcdClient Init Error")
        panic(err)
    }

    // put test
    err = mce.Etcdclient.Put("putTest/mykey", "ok")
    if err != nil {
        fmt.Println("etcdClient Put Error")
        panic(err)
    }

    // put test
    err = mce.Etcdclient.Put("putTest/mykey2", "ok")
    if err != nil {
        fmt.Println("etcdClient Put Error")
        panic(err)
    }    

    // get test
    myvalue, err := mce.Etcdclient.Get("putTest/mykey")
    if err != nil {
        fmt.Println("etcdClient Get Error")
        panic(err)
    }
    fmt.Println(myvalue)

    // get prefix
    myprefix, err := mce.Etcdclient.GetPrefix("putTest")
    if err != nil {
        fmt.Println("etcdClient GetPrefix Error")
        panic(err)
    }
    fmt.Print(myprefix)    


    // del test
    err = mce.Etcdclient.Del("putTest/mykey")
    if err != nil {
        fmt.Println("etcdClient Del Error")
        panic(err)
    }

    // del test
    err = mce.Etcdclient.Del("putTest")
    if err != nil {
        fmt.Println("etcdClient Del Error")
        panic(err)
    }    

}