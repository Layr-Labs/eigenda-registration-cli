# EigenDA Registration CLI
This is a temporary CLI to generate EigenDA Registration Parameters. We plan to incorporate this in our existing tooling in near future.

## How to build
```bash
go build -v -o eigenda-registration main.go
```

## Example
```bash
./eigenda-registration --operator-address 0x2222aac0c980cc029624b7ff55b88bc6f63c538f \
      --quorums 0 \
      --bls-key-path /Users/ubuntu/.eigenlayer/operator_keys/test.bls.key.json \
      --churner-url churner-preprod-holesky.eigenda.xyz:443 \
      --bls-key-password "password"
```