## 部署文档
1. 新建数据库 ```wallet_transition_prod```
2. Go 源码编译
3. 迁移私钥到服务器 A
4. 启动 ```wallet_middle``` 服务
5. 启动 ```wallet_core``` 服务 (部署在服务器 A)
6. 启动 ```wallet_gateway```

3,4,5,6 均使用放在当前服务器用户目录下的 ```wallet-transition.yml``` 文件，配置文件内容均不同，下文会给出。

> **注意：服务 ```wallet_gateway``` 要在 ```wallet_middle``` 和 ```wallet_core``` 之后启动**

### 迁移节点私钥到独立签名服务器
分两步：
1. 从节点导出私钥 (本地操作)
2. 导入私钥到签名服务器 (签名服务器上操作)

#### 从节点导出私钥 (本地操作)
配置文件放置在本地当前用户目录下，```wallet-transition.yml``` 内容：
```yml
# BTC node info
btc_node_host:  "192.168.12.100:18443"
btc_node_usr:   "btcrpc"
btc_node_pass:  "123456"
btc_http_mode:  true
btc_disable_tls: true

# new wallet server
new_wallet_server_host: "192.168.12.101:22"
new_wallet_server_user: "root"
new_wallet_server_pass: "vj!#*Lianxi@2018"

# btc wallet server info
old_btc_wallet_server_host: "192.168.12.100:22"
old_btc_wallet_server_user: "root"
old_btc_wallet_server_pass: "vj!#*Lianxi@2018"

# Ethereum wallet info
old_eth_wallet_server_host: "192.168.12.101:22"
old_eth_wallet_server_user: "root"
old_eth_wallet_server_pass: "vj!#*Lianxi@2018"
keystore_path: "/home/eth/privnet/data1/keystore"
ks_pass: ""
```
根据不同平台编译可执行文件 ```wallet_tools```，修改配置文件中比特币旧节点的信息、新旧服务器的登录信息之后执行如下命令：
```shell
wallet-transition|master⚡ ⇒ wallet_tools dump -a btc
{"level":"info","msg":"Using Configure file: /Users/lianxi/wallet-transition.yml Time: Tue Jan 15 13:50:59 2019"}
{"level":"info","msg":"dump old btc wallet result: success"}
{"level":"info","msg":"Copy to new server: 192.168.12.100:22:/usr/local/wallet-transition/btc.backup_new"}

wallet-transition|master⚡ ⇒ wallet_tools dump -a eth
{"level":"info","msg":"Using Configure file: /Users/lianxi/wallet-transition.yml Time: Tue Jan 15 13:57:54 2019"}
{"level":"info","msg":"Ethereum account: 0xB642728F152Fd6935425925adaD25f1311020880"}
{"level":"warn","msg":"Keystore DecryptKey error: could not decrypt key with given passphrase"}
{"level":"info","msg":"Copy to new server: 192.168.12.101:22:/usr/local/wallet-transition/eth.backup_new"}
```
#### 导入私钥到独立签名服务器 (签名服务器上操作)
编译好 ```wallet_tools``` 的源码之后，把可执行文件复制 (scp) 签名服务器，在当前用户下的配置文件 ```wallet-transition.yml``` 内容为:
```yml
// 根据数据库配置修改
db_mysql: "root:12345678@tcp(localhost:32781)/wallet_transition_dev"
```
执行如下命令导入到签名服务器：
```shell
wallet_tools migrate -a btc // 导入 btc 私钥
wallet_tools migrate -a eth // 导入 eth 私钥
```

### 区块链节点启动配置修改
#### 比特币
- 把 ```wallet_gateway``` 和 ```wallet_middle``` 的 ip 加到 ```rpcallowip```
- 在 -conf 的配置文件中加上 ```wallet_middle``` 的 ```endpoint```

  ```blocknotify=curl http://192.168.12.101:3001/btc-best-block-notify?hash=%s```

#### 以太坊
- ```--rpcapi``` 加上 ```txpool```
- ```--txpool.accountqueue 1000```

### wallet_middle 服务
在启动之前，要修改 bitcoind 的配置文件，在 -conf 的文件中加入：
```shell
// IP 要改成 wallet_middle 所在服务的内网 ip
blocknotify=curl http://192.168.12.101:3001/btc-best-block-notify?hash=%s
```
wallet_middle 服务要使用到的 ```wallet-transition.yml``` 内容格式如下，内容要根据配置的信息做修改：
```yml
# BTC node info
#btc_node_host:  "192.168.12.100:18443"
btc_node_host:  "127.0.0.1:8332"
btc_node_usr:   "btcrpc"
btc_node_pass:  "123456"
btc_http_mode:  true
btc_disable_tls: true

db_mysql: "root:12345678@tcp(192.168.12.127:32781)/wallet_transition_staging"
```
### wallet_core 独立签名服务
> 签名服务原则上要独立部署在安全的内网服务器上，并且 ```～/.db_wallet``` 文件夹要定时备份。私钥一旦泄漏或丢失，损失的数字资产将不可追回。

```wallet_core ``` 服务使用的配置文件如下：
```shell
wallet-transition|master⚡ ⇒ cat ~/wallet-transition.yml
wallet_core_rpc_url: "localhost:50051"
```
### wallet_gateway 外部接口服务
该服务放在最后启动。配置文件格式如下，内容要做对应修改：
```yml
wallet-transition|master⚡ ⇒ cat ~/wallet-transition.yml
# BTC node info
btc_node_host:  "192.168.12.100:18443"
btc_node_usr:   "btcrpc"
btc_node_pass:  "123456"
btc_http_mode:  true
btc_disable_tls: true
# Ethereum node info
eth_rpc: "http://127.0.0.1:8545"

/* wallet_core_rpc_url 要改成 wallet_core 服务所在的 ip */
wallet_core_rpc_url: "{wallet_core_ip}:50051"

api_assets: ["btc", "eth", "abb", "abb2", "sb", "omni_second_token"]
eth_token:
    #"abb": "0x9ac793a28d5207ce2ddd41542dbf5363d68324a8" local
    "abb": "0x55c725f4bd2a94749147f9c5af09f9fd9e765cc7"
    "abb2": "0x742baf6067b702f832a13a492b5a00f44cc28dd7"
    "sb": "0xb9c9599d16c05d3abb13e84fd5e2348e3861e386"
confirmations:
    "btc": 6

db_mysql: "root:12345678@tcp(localhost:32781)/wallet_transition_dev"
```
### 其他
目前 Go 源码需要 docker 服务跨平台编译，以后 ```wallet_middle```, ```wallet_core``` 和 ```wallet_gateway``` 三个服务要 Docker 化自动部署。
