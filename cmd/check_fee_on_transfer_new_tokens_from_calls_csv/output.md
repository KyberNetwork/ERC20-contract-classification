## Tokens

* FoT (taxSell, taxBuy) + Rebase: https://etherscan.io/address/0x9b0e1c344141fb361b842d397df07174e1cdb988#code
* FoT (burnTax - i.e charge fee on transfer, but fee is burnt): https://etherscan.io/address/0xe4ab0be415e277d82c38625b72bd7dea232c2e7d#code
* FoT (dynamic fee percent depends on number buys/sells): https://etherscan.io/token/0x6580685617a8721df77ca42a08e7b1d58da79cf9#code
* Normal (no tax, no rebase, no proxy): https://etherscan.io/address/0x04c17b9d3b29a78f7bd062a57cf44fc633e71f85#code

## Output

```
checking token 0x9b0E1C344141fB361B842d397dF07174E1CDb988
numScenarios = 10000
    numEqual = 4853, numLess = 3469
    token 0x9b0E1C344141fB361B842d397dF07174E1CDb988 is fee-on-transfer
```

```
checking token 0xE4aB0bE415e277d82C38625B72BD7DeA232C2E7d
numScenarios = 10000
    numEqual = 5753, numLess = 3015
    token 0xE4aB0bE415e277d82C38625B72BD7DeA232C2E7d is fee-on-transfer
```

```
checking token 0x6580685617A8721dF77Ca42A08E7B1d58dA79CF9
numScenarios = 644
    numEqual = 1, numLess = 502
    token 0x6580685617A8721dF77Ca42A08E7B1d58dA79CF9 is fee-on-transfer
```

```
checking token 0x04C17b9D3b29A78F7Bd062a57CF44FC633e71f85
numScenarios = 10000
    numEqual = 5372, numLess = 0
    token 0x04C17b9D3b29A78F7Bd062a57CF44FC633e71f85 is NOT fee-on-transfer
```
