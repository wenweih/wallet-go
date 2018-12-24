## 钱包服务独立模块架构设计
### 服务类型
0. 区块链节点服务
  - 开放 JSON-RPC 接口
1. 钱包辅助工具
  - 从节点导出钱包
2. 钱包核心服务
  - 生成钱包
  - 钱包签名
3. 钱包对外接口
  - 生成钱包接口
  - 提现「构建原始交易、广播签名后的交易」
  - 获取区块信息
  - 获取交易信息
4. 钱包辅助服务
  - 同步监听 UTXO (UTXO base blockchain, such as bitcoin)

### 业务流程
![image](./img/wallet-module-business-logic.png)
### 服务体系结构
![image](./img/wallet-module-architecture.png)
