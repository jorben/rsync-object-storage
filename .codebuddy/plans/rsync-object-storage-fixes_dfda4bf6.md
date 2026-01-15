---
name: rsync-object-storage-fixes
overview: 修复8个架构问题（优雅退出、channel管理、锁优化、协程泄漏、废弃代码、全局状态说明），并给出其余问题的改进方案文档。
todos:
  - id: explore-codebase
    content: 使用 [subagent:code-explorer] 探索项目结构，定位所有13个问题的具体代码位置和上下文
    status: completed
  - id: fix-graceful-shutdown
    content: 修复问题1：实现优雅退出机制，添加信号处理和资源清理逻辑
    status: completed
    dependencies:
      - explore-codebase
  - id: fix-channel-management
    content: 修复问题2：优化channel管理，确保正确创建、使用和关闭
    status: completed
    dependencies:
      - explore-codebase
  - id: fix-lock-optimization
    content: 修复问题3和4：优化锁机制，修复协程泄漏问题
    status: completed
    dependencies:
      - explore-codebase
  - id: fix-deprecated-code
    content: 修复问题7、8、10：清理废弃代码，修复相关架构问题
    status: completed
    dependencies:
      - explore-codebase
  - id: fix-global-state
    content: 修复问题13：为全局单例添加设计说明和mock方法注释
    status: completed
    dependencies:
      - explore-codebase
  - id: create-improvement-doc
    content: 创建改进方案文档，包含问题5、6、9、11、12的详细改进方案
    status: completed
    dependencies:
      - explore-codebase
---

## Product Overview

针对 rsync-object-storage 项目的架构问题修复计划，包含8个需要直接修复的问题和5个需要提供改进方案文档的问题。

## Core Features

### 需要修复的问题（8个）

- **问题1**: 优雅退出机制修复
- **问题2**: Channel管理优化
- **问题3**: 锁机制优化
- **问题4**: 协程泄漏修复
- **问题7**: 废弃代码清理
- **问题8**: 相关架构问题修复
- **问题10**: 相关架构问题修复
- **问题13**: 全局单例模式保留，添加mock说明注释

### 需要提供改进方案的问题（5个）

- **问题5**: 改进方案文档
- **问题6**: 改进方案文档
- **问题9**: 改进方案文档
- **问题11**: 改进方案文档
- **问题12**: 改进方案文档

### 交付物

- 修复后的代码文件
- 改进方案文档（IMPROVEMENT_PROPOSALS.md）

## Tech Stack

- 语言: Go
- 项目类型: CLI工具 / 对象存储同步工具

## 实现细节

### 修复范围分析

根据问题类型，需要修改的核心模块：

#### 1. 优雅退出机制（问题1）

- 确保所有goroutine能正确响应context取消信号
- 添加shutdown hook和清理逻辑
- 使用sync.WaitGroup确保所有工作完成

#### 2. Channel管理（问题2）

- 检查所有channel的创建、使用和关闭
- 确保channel在适当时机关闭
- 避免向已关闭channel发送数据

#### 3. 锁优化（问题3）

- 减少锁持有时间
- 考虑使用sync.RWMutex替代sync.Mutex
- 避免锁嵌套导致的死锁风险

#### 4. 协程泄漏修复（问题4）

- 确保所有goroutine有退出条件
- 使用context传递取消信号
- 添加超时机制

#### 5. 废弃代码清理（问题7）

- 移除未使用的函数和变量
- 清理注释掉的代码块
- 移除过时的依赖

#### 6. 全局状态说明（问题13）

- 保持全局单例模式
- 添加详细注释说明设计意图
- 提供测试时mock的方法说明

### 核心目录结构（仅显示需修改的文件）

```
rsync-object-storage/
├── main.go                    # 优雅退出入口点
├── internal/
│   ├── sync/
│   │   ├── worker.go          # 协程管理、channel处理
│   │   └── pool.go            # 工作池、锁优化
│   ├── storage/
│   │   └── client.go          # 全局单例注释
│   └── utils/
│       └── deprecated.go      # 废弃代码清理
├── docs/
│   └── IMPROVEMENT_PROPOSALS.md  # 改进方案文档
```

### 关键代码结构

**优雅退出模式**

```
// 优雅退出处理
func gracefulShutdown(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    select {
    case <-sigChan:
        cancel()
    case <-ctx.Done():
    }
    
    wg.Wait()
}
```

**全局单例mock说明注释模板**

```
// GlobalClient 是存储客户端的全局单例实例
// 设计说明：采用单例模式简化客户端管理，避免重复创建连接
// 
// 测试时Mock方法：
// 1. 使用接口替换：定义 StorageClientInterface，GlobalClient 实现该接口
// 2. 在测试中使用 mock 实现替换：
//    var mockClient = &MockStorageClient{}
//    originalClient := GlobalClient
//    GlobalClient = mockClient
//    defer func() { GlobalClient = originalClient }()
var GlobalClient *StorageClient
```

### 改进方案文档结构

```markdown
# IMPROVEMENT_PROPOSALS.md

## 问题5: [问题标题]
### 现状分析
### 改进方案
### 实施步骤
### 预期收益

## 问题6: [问题标题]
...
```

## Agent Extensions

### SubAgent

- **code-explorer**
- Purpose: 探索项目代码库，定位需要修复的具体文件和代码位置，理解现有架构和问题上下文
- Expected outcome: 获取所有13个问题的具体代码位置、现有实现方式，为修复工作提供准确的代码上下文