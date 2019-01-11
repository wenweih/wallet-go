### Generate Token.go
```shell
solc --abi EIP20Interface.sol -o ./
abigen --abi=EIP20Interface.abi --pkg=token --out=erc20.go
// then modify erc20.go package as blockchain
```
