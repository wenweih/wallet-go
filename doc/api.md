## 钱包模块对外接口文档
#### 例子

```shell
/* RSA 公钥加密参数：
{
  "asset": "btc",
  "detail": {
    "from": "mincESMHycRwGihfLwQGSZNwXwyRDtfzPZ",
    "to": "mqqF7ncKVdF3FTPGv4P7ZhzNpK97aDV1Z2",
    "amount": 1.5
  }
}

得到密文作为请求头的 Authorization 参数值，且请求类型为：application/json
*/

curl -i -X POST http://localhost:3000/withdraw -H "Content-Type: application/json" -H "Authorization: 5834eff8eb2046de1ac3a2eba688ac2e217ceccf06d76270bfdbc3decd207b049977ac4e3896ccd9233340a652cb5c98ad4f43de1654040f329e0ff8c1c1fd29796a40a15fdcbfc2d591c0a24dbf32af2f178c0ba29c7c981346ef9d70411814d86fe594e0d1299bc96d96515948ba9f32b1fe3a1ca8a811b147510e34462c3682bab22c9dc15f9c1b79bc27239be621a05224ce0c219e727046bd10c1c52d549369b6a387edf01006a46c61a9c4fb1e0b0cf1e794d535f716b05ed05ec04d6a1d67497c5f34607e5a7aee3bd1c2f6330c42641bdb91f87691d2ac03e4350ecd34a3580f6a48cfac4d28e42a46c2a256c1c075b443deac5b85b0bd1730b2982a2bcf638d091e1aba09cf3847cbd0859db0c04270ffbcb99fef731ad7762cd05f412a6524afe00e5f98749dc262ec0f1b789af12fc8cbb092581bf25de0b1a3e646502bdee0e5cfcaa616bd9b90a91acc8b4cd002c7e77197d604e1b1ad4f5e3796e0fb1622a03f74636b5d059879b7ec0a41e9a58f6e2effa0ffdc1cb49cd1feb13f972127678549e22424816ad10a80adda290731164ae234a4058ec2e78fd1f8cfc4b5bc15877f4dc914a2beb574d41bb6c7e833403881103d2ea4497720740d88204d1c4ac977a7ec15741542457dfea4189c5934bfeb408aa29fd9c7cb1e31cfbe814a8e24d4e2b741dee852a8a259f339a0be731207ca63e65c7a13cc11"
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Date: Thu, 20 Dec 2018 01:29:40 GMT
Content-Length: 1716

{"status":200,"txid":"0x55b65229c187352b19f9e31cccc1a81a694d1945b48e23d42469e861d30e9e95"}
```

#### 状态码
|Code|Desc|
|---|---|
|404|请求错误|
|401|鉴权失败|
|404|禁止访问|
|404|资源不存在|
|500|服务内部错误|
|200|响应成功|

#### 服务端返回
服务端响应失败返回状态码和错误消息：
```json
{
  "code":500,
  "msg":"Bitcoin SendRawTransaction signed tx error:-27: transaction already in block chain"
}

{
  "code":403,
  "msg":"Decode Token error"
}
```
服务端响应成功返回状态码和具体接口对应的字段，比如请求生成 ETH 地址的接口：
```shell
wallet-go|master⚡ ⇒ curl -i -X POST http://localhost:3000/address -H "Content-Type: application/json" -H "Authorization: b3d9b7de0ba463f722ffda964877de2d6d8c6bba86defb8d603bdba3548c716bbfe310283a6e86127c7259e5d124b9d55263c374be78b2e72f4dd4045eff90670a42ba2ad1e650f26e7baa92ab0ada1e471572e80e50cb0138f15178f29fb759ff436d87814df1ea48ff1fbd0b7a32dacbdc06fc76b17b4c60e0c6d278b17523b9218e7805c56ac80d3027072fdcb3bb6df33a48a37ebc6a68a0bd26af287358acfa2653eb3380dd34c62a027a73f76419c5b949f2d83508c89943820ac3f89f0a0169d6d6351ccbcda0daec60cf4761287b4e74ca0c4e2a392821de36bf2bcb36f00d770eb097793b20a947b4769fea9cd9c93c4a1d2b65f513c3feaa1bd84f1821cbd070291b5d27ecff5ae1faaf9bf390c14a1b4b27c6a1e99f730582a2afc451b9b8ea39b91b79d680cc8df2c60189d591d9732f2853466bc24032442ff7a93e6a7a30d9ba7acbc44a8fca12ae98ec3eb8d7be80197ab7174eb6c33cf8d1ae48eda8ba7b0bb0517b956e5c5ce2cb7a8df153ba1c4f47882603eb28061d738973f8deb27b14810057719de1e15c8ff0323b1afbb3e0b97b7cf88e71d6a9c4207a8465ef52ed9d0e5f7b1961ffe265821670de46c54de68d3025bb64f6f7935d254e207271f36945e9035d250346c2043d65d976e88568ea6a73fd2223100f4b7f1e8889e3e04f2d8c4c9b13c9a8d633a093e90df8dc570d76d9b30a0c00fb"
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Date: Thu, 20 Dec 2018 02:11:29 GMT
Content-Length: 69

{
  "address":"0x9B6B0802d41164Fc05c7F17260c8377F080cEB79",
  "status":200
}
```
#### 接口
- 生成地址 ```POST /address```
- 提现 ```POST /withdraw```
- 获取区块信息  ```POST /block```
- 获取交易信息  ```POST /tx```

测试 RSA public key
```shell
-----BEGIN RSA PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAthER5GVP+ZT5jH67cjmz
70G1oUzk7VqAONCbukx8qey9cvEp4pd/YajffSPNHavaAQZIevA9QRCQjrJH2adV
gFdoLRndkKwy08FEtet8MNKb+LzImRUW1b4gtQqy0CuKoxdkY36FyDFp1L+at55t
3A46CNAkCq4KVsH39Q+RB/NqcPyMk0l8breAyO4rqSbcUBFr41yAZ/3V8YDPK/xE
FNRkscZwnkcfnewmauwn9NmlPoPbkgrDmeWHE8su67Z6833FoLEWNRbCXbik2JSx
4MEY7PvntaPVkvI8SR3g/ShdZTDNSTU6hlsBG0hOO9bPGfblj9mqUJiXLL/FWMb2
F0jmWIuH18JQ0SbyT07/AlseMVSKePIPqkih5nElatd2/D+fEwrDD7YlqEG+N0E5
//nUlKn912W7GF8ob6j87gnKZba/rxuf8ZmyklJ8MKv0B8hXOk9fXAIWKbRrWX0l
3LRpoyuxkZfp3yYj6wsUd8LPkX635SnJVwuB9kIEN6gSsGzvZZRXwxQhQVlUOVs+
kEPNAiR5+v7f0bHdpZ9J1M0+kQVOfw6IgzhwcDLgNTjyFouw6VEW72BkPjSjziwz
xAmyKoHheymmUud6fcATq3K6pwNKlUAqqf6nKwZQ6IvvQH/D2uGCKuEcudjcayGH
+vxYE9cwdQIxWDEfdMFoqD8CAwEAAQ==
-----END RSA PUBLIC KEY-----
```
- 鉴权：```-H "Authorization: {rsa 公钥加密参数密文}```
- 请求类型：```-H "Content-Type: application/json"```


> 备注：
1. 请求参数不是放在 body ，而是加密后作为请求头的 Authorization 参数值。
2. 请求类型为：Content-Type: application/json


##### 生成地址
- URL: ```/address```
- Method: ```POST```
- Params:
  ```json
  {
    "asset": "btc" // btc or eth
  }
  ```
- Response:
  ```json
  {
    "address":"0x9B6B0802d41164Fc05c7F17260c8377F080cEB79",
    "status":200
  }
  ```

##### 提现
- URL: ```/withdraw```
- Method: ```POST```
- Params:
  ```json
  {
    "asset": "btc" // btc or eth
    "detail": {
      "from": "from_address",
      "to": "to_address",
      "amount": 1.5 // float64
    }
  }
  ```
- Response:
  ```json
  {
    "status":200,
    "txid":"0xa2d45430e723df5aada4aa36fa19eb3cea2b05953f30af8a71e4bbe04bd5ff23"
  }
  ```

### 获取区块信息
- URL: ```/block```
- Method: ```POST```
- Params:
  ```json
  {
    "asset": "btc" // btc or eth
    "detail": {
      "height": {height}  // int64
    }
  }
  ```
- Response:
  ```json
  {
    "status":200,
    "block": {
      "hash": "aaaaa"
    }
  }
  ```

### 获取交易信息
- URL: ```/tx```
- Method: ```POST```
- Params:
  ```json
  {
    "asset": "btc" // btc or eth
    "detail": {
      "txid": {txid}
    }
  }
  ```
- Response:
  ```json
  {
    "status":200,
    "tx": {
      "hash": "aaaaa"
    }
  }
  ```
