package cursor

// GlobalDBReader Global 数据库读取器接口
// 用于从 Cursor Global 数据库读取数据，使用只读模式，支持 WAL
type GlobalDBReader interface {
	// ReadValue 从 Global 数据库读取指定键的值
	// key: 要查询的键，如 "aiCodeTracking.dailyStats.v1.5.2024-01-15"
	// 返回: 对应的值（BLOB 数据）和错误
	ReadValue(key string) ([]byte, error)
}
