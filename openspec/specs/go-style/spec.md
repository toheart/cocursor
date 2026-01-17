# Go 编码规范

本规范定义 cocursor 项目的 Go 代码编写标准，基于 [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)。

## 基础规范

- 使用 `golangci-lint` 进行代码检查
- 注释使用中文
- 日志使用英文，需符合日志等级
- 变量命名使用 camelCase，导出变量使用 PascalCase

## Uber Go Style Guide 核心规范

### 1. 接口合规性验证

使用编译时检查确保类型实现接口：

```go
var _ http.Handler = (*Handler)(nil)
```

### 2. 零值 Mutex 有效

不需要初始化指针：

```go
var mu sync.Mutex  // 正确
mu := new(sync.Mutex)  // 避免
```

### 3. 边界处复制 Slice/Map

接收或返回时创建副本，防止外部修改：

```go
func (d *Driver) SetTrips(trips []Trip) {
    d.trips = make([]Trip, len(trips))
    copy(d.trips, trips)
}
```

### 4. 使用 defer 清理资源

文件、锁等资源使用 defer 释放：

```go
p.Lock()
defer p.Unlock()
```

### 5. Channel 大小为 1 或无缓冲

避免使用任意大小的缓冲 channel。

### 6. 枚举从 1 开始

避免零值歧义：

```go
const (
    Add Operation = iota + 1
    Subtract
)
```

### 7. 错误处理规范

- 使用 `pkg/errors` 包装错误，返回错误而非 panic
- 错误只处理一次，不要同时 log 和 return
- 使用 `%w` 包装错误以支持 `errors.Is/As`
- 错误变量使用 `Err` 前缀，错误类型使用 `Error` 后缀

### 8. 不要 Panic

生产代码避免 panic，返回 error：

```go
func run(args []string) error {
    if len(args) == 0 {
        return errors.New("an argument is required")
    }
    return nil
}
```

### 9. 避免可变全局变量

使用依赖注入替代。

### 10. 避免 init()

除非必要，将初始化逻辑放在 main() 或构造函数中。

### 11. 避免 goroutine 泄漏

每个 goroutine 必须有可预测的退出时机。

## 性能规范

- 优先使用 `strconv` 而非 `fmt` 进行类型转换
- 避免重复的 string-to-byte 转换
- 初始化 map/slice 时指定容量

## 风格规范

- 软行长度限制 99 字符
- 使用字段名初始化结构体
- 省略结构体中的零值字段
- nil 是有效的空 slice，使用 `len(s) == 0` 检查空
- 减少嵌套，提前返回处理错误
- 导出函数放在文件顶部，按调用顺序排列
- 未导出的全局变量使用 `_` 前缀
