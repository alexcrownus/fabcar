# fabcar

[Hyperledger Fabric](https://www.hyperledger.org/projects/fabric) [tutorial](https://hyperledger-fabric.readthedocs.io/en/latest/write_first_app.html) nodejs code ported to [Go](https://golang.org) using [fabric-sdk-go](https://github.com/hyperledger/fabric-sdk-go).

#### SoftHSM 
* Install softhsm
    
    ```sh 
    brew install softhsm
    
* Init softhsm

    ```sh
    softhsm2-util --init-token --slot 0 --label "ForFabric" --so-pin 1234 --pin 98765432