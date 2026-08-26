# Hyperliquid Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/funcblock-quant/hyperliquid-go-sdk.svg)](https://pkg.go.dev/github.com/funcblock-quant/hyperliquid-go-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/funcblock-quant/hyperliquid-go-sdk)](https://goreportcard.com/report/github.com/funcblock-quant/hyperliquid-go-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Hyperliquid Go SDK 是一个功能完整的 Go 语言 SDK，为 [Hyperliquid](https://hyperliquid.xyz) 去中心化永续期货交易所提供全面的 API 支持。

## 📋 目录

- [特性](#特性)
- [安装](#安装)
- [快速开始](#快速开始)
- [功能说明](#功能说明)
- [API 参考](#api-参考)
- [示例](#示例)
- [环境配置](#环境配置)
- [测试](#测试)
- [贡献](#贡献)


## ✨ 特性

### 🔌 核心功能

- **多环境支持**: 支持主网和测试网
- **HTTP 客户端**: 高性能 HTTP 客户端，支持超时控制
- **WebSocket 客户端**: 实时数据订阅，支持自动重连
- **类型安全**: 完整的类型定义和结构体映射
- **错误处理**: 完善的错误处理和类型化错误响应

### 📊 市场数据

- **实时价格**: 获取所有交易对的中间价格
- **订单簿**: L2 深度数据查询
- **K线数据**: 支持多种时间间隔的历史 K 线数据
- **交易历史**: 用户交易记录和成交明细
- **资金费率**: 历史资金费率和用户资金费用记录

### 💼 交易功能

- **订单管理**: 限价单、市价单、触发单
- **批量操作**: 批量下单、取消、修改订单
- **杠杆设置**: 调整交易对杠杆倍数（全仓/逐仓）
- **保证金管理**: 逐仓保证金调整
- **减仓功能**: 安全的仓位管理

### 🔐 安全与签名

- **以太坊签名**: 完整的 EIP-712 签名支持
- **私钥管理**: 安全的私钥处理和签名生成
- **Nonce 管理**: 自动的 nonce 管理和时间戳同步

### 💰 资产管理

- **用户状态**: 查询账户余额、持仓和保证金信息
- **转账功能**: USD 转账和 Vault 操作
- **资金历史**: 充值、提现和资金费用记录
- **现货交易**: 支持现货市场操作

### 📈 高级功能

- **实时订阅**: WebSocket 实时数据订阅
- **推荐系统**: 推荐状态查询
- **质押功能**: 质押奖励和委托管理
- **子账户**: 多账户管理支持

## 🚀 安装

确保您的 Go 版本为 1.24.0 或更高版本：

```bash
go mod init your-project
go get github.com/funcblock-quant/hyperliquid-go-sdk
```

### 依赖要求

- Go 1.24.0+
- `github.com/ethereum/go-ethereum` - 以太坊相关功能
- `github.com/gorilla/websocket` - WebSocket 支持
- `github.com/vmihailenco/msgpack/v5` - 消息序列化

## 🚀 快速开始

### 基本市场数据查询

```go
package main

import (
    "fmt"
    "log"
    
    sdk "github.com/funcblock-quant/hyperliquid-go-sdk"
)

func main() {
    // 创建信息客户端
    info, err := sdk.NewInfo(sdk.MainnetAPIURL)
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取所有交易对价格
    mids, err := info.AllMids()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("BTC 价格: %s\n", mids["BTC"])
    fmt.Printf("ETH 价格: %s\n", mids["ETH"])
}
```

### 交易示例

```go
package main

import (
    "log"
    
    sdk "github.com/funcblock-quant/hyperliquid-go-sdk"
)

func main() {
    // 从调用方配置系统读取私钥后传入 SDK；私钥不包含 0x 前缀
    signer, err := sdk.NewLocalSignerFromHex("your_private_key_without_0x_prefix")
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取元数据
    info, err := sdk.NewInfo(sdk.MainnetAPIURL)
    if err != nil {
        log.Fatal(err)
    }
    meta, err := info.Meta()
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建交易客户端
    exchange := sdk.NewExchange(sdk.MainnetAPIURL, nil, meta, signer)
    
    // 下限价单
    orderReq := sdk.OrderRequest{
        Coin:    "BTC",
        IsBuy:   true,
        Size:    0.01,
        LimitPx: 40000.0,
        OrderType: sdk.OrderType{
            Limit: &sdk.LimitOrderType{Tif: sdk.TifGtc},
        },
    }
    
    result, err := exchange.Order(orderReq, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("订单结果: %+v", result)
}
```

### WebSocket 订阅

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    sdk "github.com/funcblock-quant/hyperliquid-go-sdk"
)

func main() {
    // 创建 WebSocket 客户端
    ws := sdk.NewWebsocketClient(sdk.MainnetAPIURL)
    
    // 连接
    ctx := context.Background()
    if err := ws.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer ws.Close()
    
    // 订阅 BTC 交易数据
    sub := sdk.Subscription{
        Type: sdk.SubTypeTrades,
        Coin: "BTC",
    }
    
    _, err := ws.Subscribe(sub, func(data interface{}) {
        fmt.Printf("BTC 交易数据: %+v\n", data)
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 保持连接
    select {}
}
```

## 📚 功能说明

### Info 客户端 - 市场数据和账户信息

Info 客户端覆盖市场数据、用户账户、资金费率、Vault、dex 维度查询和 TWAP 等只读接口。

```go
info, err := sdk.NewInfo(sdk.MainnetAPIURL)
infoWithDexs, err := sdk.NewInfoWithPerpDexs(sdk.MainnetAPIURL, []string{"", "testDex"})

// 市场数据
mids, err := info.AllMids()                    // 所有交易对价格
meta, err := info.Meta()                       // 交易对元数据
l2Book, err := info.L2Snapshot("BTC")          // 订单簿快照
candles, err := info.CandlesSnapshot("BTC", "1h", startTime, endTime)
predictedFundings, err := info.PredictedFundings()

// 用户数据
userState, err := info.UserState("0x...")     // 用户状态
openOrders, err := info.OpenOrders("0x...")   // 未成交订单
fills, err := info.UserFills("0x...")         // 成交记录
fundingHistory, err := info.UserFundingHistory("0x...", startTime, nil)

// Dex 维度查询
dexMeta, err := infoWithDexs.MetaForDex("testDex")
dexMids, err := infoWithDexs.AllMidsForDex("testDex")
dexUserState, err := infoWithDexs.UserStateForDex("0x...", "testDex")
```

其他常用查询能力：

- `MetaForDex`、`AllMidsForDex`、`UserStateForDex`、`OpenOrdersForDex`、`FrontendOpenOrdersForDex`、`UserFillsForDex`、`MetaAndAssetCtxsForDex`
- `PerpDexs`、`PerpsAtOpenInterestCap`、`HistoricalOrders`、`VaultDetails`、`UserVaultEquities`、`UserRole`、`MaxBuilderFee`
- `UserRateLimit`、`UserTwapHistory`、`UserTwapSliceFills`、`PortfolioMarginUserState`

### Exchange 客户端 - 交易操作

Exchange 客户端覆盖下单、撤单、改单、批量操作、TWAP、风险控制、杠杆和保证金管理。

```go
exchange := sdk.NewExchange(apiURL, vaultAddr, meta, signer)
expiresAfter := uint64(1710000030000)
cancelTime := uint64(1710000600000)

// 订单操作
result, err := exchange.Order(orderReq, nil)           // 下单
openResult, err := exchange.MarketOpen("BTC", true, 0.01, marketPrice, 0.01, nil, nil)
closeResult, err := exchange.MarketClose("BTC", nil, marketPrice, 0.01, nil, nil)
cancelResult, err := exchange.Cancel(cancelReq)        // 取消订单
modifyResult, err := exchange.ModifyOrder(modifyReq)   // 修改订单

// 批量操作
results, err := exchange.BulkOrders(orders, nil)       // 批量下单
groupedResults, err := exchange.BulkOrdersWithGrouping(orders, nil, sdk.GroupingTpsl)
cancelResults, err := exchange.BulkCancel(cancelReqs)  // 批量取消

// TWAP 和风险控制
twapResult, err := exchange.TwapOrder(twapReq)
twapCancelResult, err := exchange.TwapCancel("BTC", twapID)
err = exchange.ScheduleCancel(&cancelTime)
exchange.SetExpiresAfter(&expiresAfter)

// 杠杆和保证金
err = exchange.UpdateLeverage("BTC", true, 10)         // 更新杠杆
err = exchange.UpdateIsolatedMargin("BTC", 1000.0)     // 调整逐仓保证金
err = exchange.TopUpIsolatedOnlyMargin("BTC", "10")    // 补充 isolated-only margin
```

订单分组支持 `na`、`normalTpsl`、`positionTpsl`。`ScheduleCancel` 可用于 dead man's switch 风险控制，`SetExpiresAfter` 可为 L1 action 设置过期时间。

### 用户签名和资金通道

SDK 根包提供统一的 `user_signed_actions` 层，用于集中管理 EIP-712 用户签名 action。

```go
// 提现和转账
withdrawResult, err := exchange.Withdraw3("0x...", "10.0")
usdSendResult, err := exchange.USDSend("0x...", "10.0")
usdTransferResult, err := exchange.USDTransfer("0x...", "10.0")
spotSendResult, err := exchange.SpotSend("0x...", "PURR", "1.0")
classTransferResult, err := exchange.USDClassTransfer("10.0", true)

// 资产通道和子账户资金划转
sendAssetResult, err := exchange.SendAsset("0x...", "", "spot", "USDC", "10.0", "")
subTransferResult, err := exchange.SubAccountTransfer("0x...", true, 10)
subSpotTransferResult, err := exchange.SubAccountSpotTransfer("0x...", true, "PURR", "1.0")

// 授权、质押和推荐
agentResult, err := exchange.ApproveAgent(agentAddress, "agent-name")
builderFeeResult, err := exchange.ApproveBuilderFee(builderAddress, "0.001%")
delegateResult, err := exchange.TokenDelegate(validatorAddress, wei, false)
referrerResult, err := exchange.SetReferrer(referrerCode)
```

用户签名 action 会自动注入 `signatureChainId = "0x66eee"` 和 `hyperliquidChain = Mainnet/Testnet`。

### 子账户、Multi-sig 和账户抽象

```go
subAccountResult, err := exchange.CreateSubAccount(name)

signersJSON, err := sdk.CanonicalMultiSigSigners(signers)
multiSigResult, err := exchange.ConvertToMultiSigUser(signersJSON)
multiSigResultWithSigners, err := exchange.ConvertToMultiSigUserWithSigners(signers)
signature, err := exchange.SignMultiSigAction(multiSigUser, action, nonce)
result, err := exchange.MultiSig(multiSigUser, outerSigner, action, signatures)

userDexResult, err := exchange.UserDexAbstraction(userAddress, true)
userSetResult, err := exchange.UserSetAbstraction(userAddress, "abstraction")
agentDexResult, err := exchange.AgentEnableDexAbstraction()
agentSetResult, err := exchange.AgentSetAbstraction("abstraction")
bigBlocksResult, err := exchange.UseBigBlocks(true)
```

`CanonicalMultiSigSigners` 用于生成稳定的 multi-sig signer JSON。Multi-sig 已提供通用封装，生产使用前建议先用测试网 multi-sig 账户做端到端验证。

### WebSocket 客户端 - 实时数据

WebSocket 客户端支持公开市场数据、用户私有事件、TWAP、BBO、spot 状态和 dex 维度订阅。

```go
ws := sdk.NewWebsocketClient(sdk.MainnetAPIURL)
nSigFigs := 5

// 市场数据订阅
ws.Subscribe(sdk.Subscription{Type: sdk.SubTypeTrades, Coin: "BTC"}, callback)
ws.Subscribe(sdk.Subscription{Type: sdk.SubTypeL2Book, Coin: "ETH"}, callback)
ws.Subscribe(sdk.Subscription{Type: sdk.SubTypeAllMids, Dex: "testDex"}, callback)
ws.Subscribe(sdk.Subscription{Type: sdk.SubTypeBBO, Coin: "BTC", NSigFigs: &nSigFigs}, callback)

// 用户数据订阅
ws.Subscribe(sdk.Subscription{Type: sdk.SubTypeUserFills, User: "0x..."}, callback)
ws.Subscribe(sdk.Subscription{Type: sdk.SubTypeOrderUpdates, User: "0x..."}, callback)
ws.Subscribe(sdk.Subscription{Type: sdk.SubTypeUserEvents, User: "0x..."}, callback)
```

`Subscription` 支持 `dex`、`nSigFigs`、`mantissa`、`aggregateByTime`、`isPortfolioMargin` 字段。常用订阅类型包括 `allMids`、`notification`、`webData2`、`webData3`、`activeAssetCtx`、`activeAssetData`、`userEvents`、`userFundings`、`userNonFundingLedgerUpdates`、`bbo`、`twapHistory`、`twapSliceFills`、`twapStates`、`spotState` 等。

### 签名、兼容性和覆盖范围

- L1 action hash 支持 `expiresAfter`，并保持旧 `SignL1Action` 调用兼容。
- 官方签名向量测试覆盖 `withdraw3` 用户签名和 `subAccountTransfer` L1 签名。
- `exchange_api` 包中已有的旧 helper 保留兼容；新增能力统一放在 SDK 根包，后续建议逐步迁移到根包接口。
- Spot/perp deploy、validator/C-signer 管理、oracle/perp dex 部署管理等长尾能力尚未补齐。

## 🔧 环境配置

### SDK 参数配置

SDK 本身不读取 `.env`，也不假设调用方项目使用哪种配置系统。作为依赖引入其他项目时，应由调用方读取环境变量、配置文件、密钥管理服务或 KMS，然后通过参数传入 SDK。

典型接入方式：

```go
package main

import (
    "log"
    "os"

    sdk "github.com/funcblock-quant/hyperliquid-go-sdk"
)

func main() {
    apiURL := sdk.TestnetAPIURL // 生产环境可显式改为 sdk.MainnetAPIURL

    privateKeyHex := os.Getenv("HL_PRIVATE_KEY")
    signer, err := sdk.NewLocalSignerFromHex(privateKeyHex)
    if err != nil {
        log.Fatal(err)
    }

    info, err := sdk.NewInfo(apiURL)
    if err != nil {
        log.Fatal(err)
    }
    meta, err := info.Meta()
    if err != nil {
        log.Fatal(err)
    }

    exchange := sdk.NewExchange(apiURL, nil, meta, signer)
    _ = exchange
}
```

核心参数：

- `apiURL`：显式传入 `sdk.TestnetAPIURL` 或 `sdk.MainnetAPIURL`。
- `signer`：通过 `sdk.NewLocalSignerFromHex(privateKeyHex)` 创建，或实现 SDK 的 `Signer` 接口。
- `vaultAddr`：可选 Vault 地址，没有则传 `nil`。
- `meta`：通过 `info.Meta()` 获取后传给 `sdk.NewExchange`。

### 本仓库示例和测试的环境变量

`.env` 只用于本仓库的 examples 和集成测试，方便本地运行，不是 SDK 使用要求。创建 `.env` 文件：

```bash
# 私钥 (不包含 0x 前缀)
HL_PRIVATE_KEY=your_testnet_private_key_here

# 可选: Vault 地址
HL_VAULT_ADDRESS=0x...
```

### 网络配置

```go
// 主网
info, err := sdk.NewInfo(sdk.MainnetAPIURL)

// 测试网
info, err := sdk.NewInfo(sdk.TestnetAPIURL)
```

## 🧪 测试

### 测试环境约定

本仓库测试分为两类：

- **本地单元测试**：不访问 Hyperliquid，不需要私钥，不产生账户副作用。
- **集成测试**：访问 Hyperliquid API/WebSocket。`examples` 下的测试默认面向测试网，部分用例会真实下单、撤单、开平仓、调整杠杆/保证金或提交转账请求。

运行集成测试前，先创建 `.env`：

```bash
# 私钥不包含 0x 前缀，建议使用测试网专用账户
HL_PRIVATE_KEY=your_testnet_private_key_here

# 可选：Vault 地址
HL_VAULT_ADDRESS=0x...
```

加载环境变量：

```bash
set -a
source .env
set +a
```

如果本机 Go build cache 目录权限受限，可以临时指定 `GOCACHE`：

```bash
export GOCACHE=/tmp/hyperliquid-go-sdk-gocache
```

### 本地单元测试

```bash
go test ./...
```

该命令用于验证 SDK 的本地逻辑，包括签名向量、请求构造、序列化、响应解析等。没有导出 `HL_PRIVATE_KEY` 时，`examples` 包中的私钥依赖测试会跳过。

### 测试网只读集成测试

以下命令只访问测试网只读接口，不会下单或修改账户：

```bash
go test -v ./examples -run 'Test(SimpleInfo|MetaInfo|GetAllTokens|OpenOrders|FrontendOpenOrders|UserDepositWithdrawTxs|UserFills|UserFundingHistory|UserPortfolio|UserFees|UserStatus|CandlesSnapshot)$'
```

覆盖能力包括：

- 市场元数据和中间价：`Meta`、`AllMids`、`PerpCoins`、`SpotCoins`
- 用户查询：`UserState`、`OpenOrders`、`FrontendOpenOrders`、`UserFills`
- 资金和费率：`UserDepositWithdrawTxs`、`UserFundingHistory`、`UserPortfolio`、`UserFees`
- K 线：`CandlesSnapshot`

### 测试网 WebSocket 集成测试

```bash
go test -v ./examples -run 'Test(CandleWebSocket|WsTrade)$|TestWebsocket/(trades subscription|l2book subscription)$'
```

覆盖测试网公开订阅：

- `candle`
- `trades`
- `l2Book`

用户私有 websocket 订阅，例如 `userFills`、`orderUpdates`，依赖测试期间账户是否刚好产生事件；它们适合在写入测试同时或之后单独验证。

### 测试网写入集成测试

以下命令会真实修改测试网账户状态：

```bash
go test -v ./examples -run 'Test(Orders|UpdateLeverage|UpdateIsolatedMargin|TransferUSD)$'
```

这些测试会执行：

- 限价挂单并撤单
- 市价开仓并 reduce-only 平仓
- 更新 cross/isolated 杠杆
- 建立隔离测试仓位、调整逐仓保证金、再平仓
- 提交 USDC 转账请求

注意：

- 即使是测试网，也建议使用专用测试账户和少量测试资金。
- `TransferUSD` 在 unified account 模式下可能被交易所拒绝，并返回 `Action disabled when unified account is active`。这表示请求已到达交易所，但当前账户模式不支持该 action。
- 不要把 `.env` 提交到 git。

### 主网测试

不建议直接使用主网执行写入测试。主网下单、撤单、杠杆、保证金和转账都是真实资金操作。

如果确实需要主网验证：

1. 使用专用低资金账户。
2. 明确把代码中的 `sdk.TestnetAPIURL` 替换为 `sdk.MainnetAPIURL`，或单独编写主网专用测试。
3. 先跑只读查询，再逐项执行写入流程。
4. 不要复用测试网私钥或生产做市主账户私钥。

### 常用单项测试

```bash
# 查询所有交易对
go test -v ./examples -run TestGetAllTokens

# 挂单、撤单、开仓、平仓
go test -v ./examples -run TestOrders

# 杠杆和逐仓保证金
go test -v ./examples -run 'Test(UpdateLeverage|UpdateIsolatedMargin)$'

# WebSocket K 线
go test -v ./examples -run TestCandleWebSocket
```

### 测试示例

项目包含丰富的测试示例：

- `examples/all_tokens_test.go` - 基础信息查询
- `examples/order_test.go` - 订单操作
- `examples/trade_test.go` - 交易功能
- `examples/websocket_test.go` - WebSocket 订阅
- `examples/candles_test.go` - K线数据
- `examples/leverage_test.go` - 杠杆操作

## 📖 API 参考

### 常用订单类型

```go
// 限价单
orderType := sdk.OrderType{
    Limit: &sdk.LimitOrderType{Tif: sdk.TifGtc},
}

// 只做市单
orderType := sdk.OrderType{
    Limit: &sdk.LimitOrderType{Tif: sdk.TifAlo},
}

// 立即成交或取消
orderType := sdk.OrderType{
    Limit: &sdk.LimitOrderType{Tif: sdk.TifIoc},
}
```

### 错误处理

```go
result, err := exchange.Order(orderReq, nil)
if err != nil {
    if apiErr, ok := err.(*sdk.APIError); ok {
        log.Printf("API 错误: %s", apiErr.Message)
    } else {
        log.Printf("其他错误: %v", err)
    }
}
```

## 🤝 贡献

欢迎贡献代码！请按照以下步骤：

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 🔗 相关链接

- [Hyperliquid 官网](https://hyperliquid.xyz)
- [Hyperliquid API 文档](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api)
- [官方 Python SDK](https://github.com/hyperliquid-dex/hyperliquid-python-sdk)

## ⚠️ 免责声明

本 SDK 为非官方实现，使用前请充分测试。交易有风险，请谨慎使用。

---

**如有问题或建议，欢迎提交 Issue 或 Pull Request！**
