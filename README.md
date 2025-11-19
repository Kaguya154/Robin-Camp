## 设计思路

### 数据库

选择`modernc.org/sqlite`数据库，完全go实现，0成本跨平台，简化部署。

### 后端服务

使用`CloudWeGo Hertz`框架，高性能，低延迟，易扩展。  
使用`Swaggo`生成API文档，同时提供OpenAPI3转换工具。


## 环境配置

### 1. 安装依赖

确保已安装 **Go 1.25.0** 或更高版本

```shell
go mod tidy
```

### 2. 配置环境变量

复制 `.env.example` 为 `.env` 并根据实际情况修改：
```shell
cp .env.example .env
```

示例

```env
PORT=8080
ADDRESS=0.0.0.0

# Authentication
AUTH_TOKEN=TOKEN

# Database Configuration (for the application, not used directly by e2e tests)
DB_URL="file:movies.db?_foreign_keys=on"

# Box Office API Integration
BOXOFFICE_URL=https://m1.apifoxmock.com/m1/7149601-6873494-default
BOXOFFICE_API_KEY=0B4nmUwMPBphsKDr_u9HX
```

若无法使用.env文件，可直接在运行命令前设置环境变量，例如：

```shell
export PORT=8080
export ADDRESS=0.0.0.0
export AUTH_TOKEN=TOKEN
export DB_URL="file:movies.db?_foreign_keys=on"
export BOXOFFICE_URL=https://m1.apifoxmock.com/m1/7149601-6873494-default
export BOXOFFICE_API_KEY=0B4nmUwMPBphsKDr_u9HX
```

### 使用 Air 热重载

```bash
# 安装 Air
go install github.com/air-verse/air@latest

# 运行 (配置文件: .air.toml)
air
```

## 生产

```shell
# 编译可执行文件
go build

# 运行
./Robin-Camp

# Windows 下
.\Robin-Camp.exe
```

## API 文档

### 生成 Swagger 文档

第一次运行：

```shell
# 安装swag
go install github.com/swaggo/swag/cmd/swag@latest
```

每次修改 API 注释后需要重新生成文档：

```shell
swag init --parseDependency --parseInternal --parseDepth 5 --instanceName "swagger"
```

### 访问 Swagger 文档

以`-swagger`参数启动服务后，从控制台输出的链接访问 Swagger 文档，例如：

```
http://localhost:8080/swagger/index.html
```


### OpenAPI3 转换

生成的 swagger 文档是 openapi2 格式，如需转换为 openapi3 格式，可使用内置转换工具：

```shell
go run ./docs/converter/main.go -i ./docs/swagger.json -o ./docs/openapi3.yml
```

## 使用 Docker 快速启动（推荐）

如果你不想在本地安装 Go 或编译代码，可以直接使用已经发布好的 Docker 镜像 `Kaguya154/robin-camp` 快速启动服务。

```bash
# 启动服务（Linux/macOS 示例）
docker run -d \
  --name robin-camp \
  -p 8080:8080 \
  -e PORT=8080 \
  -e ADDRESS=0.0.0.0 \
  -e AUTH_TOKEN=TOKEN \
  -e DB_URL="file:/data/movies.db?_foreign_keys=on" \
  -e BOXOFFICE_URL="https://m1.apifoxmock.com/m1/7149601-6873494-default" \
  -e BOXOFFICE_API_KEY="0B4nmUwMPBphsKDr_u9HX" \
  -v movies-data:/data \
  kaguya154/robin-camp:latest
```

Windows PowerShell 示例：

```powershell
$env:AUTH_TOKEN="TOKEN"
$env:DB_URL="file:/data/movies.db?_foreign_keys=on"
$env:BOXOFFICE_URL="https://m1.apifoxmock.com/m1/7149601-6873494-default"
$env:BOXOFFICE_API_KEY="0B4nmUwMPBphsKDr_u9HX"

docker run -d `
  --name robin-camp `
  -p 8080:8080 `
  -e PORT=8080 `
  -e ADDRESS=0.0.0.0 `
  -e AUTH_TOKEN=$env:AUTH_TOKEN `
  -e DB_URL=$env:DB_URL `
  -e BOXOFFICE_URL=$env:BOXOFFICE_URL `
  -e BOXOFFICE_API_KEY=$env:BOXOFFICE_API_KEY `
  -v movies-data:/data `
  kaguya154/robin-camp:latest
```

启动后可以通过浏览器访问：

```text
http://localhost:8080
```

## 使用发布二进制部署

如果你不想在本地安装 Go 或编译代码，也可以直接使用已经发布好的二进制文件快速启动服务。

1. 前往 [Releases 页面](https://github.com/Kaguya154/Robin-Camp/releases) 下载对应平台的压缩包（例如 `robin-camp-vX.Y.Z-linux-amd64.tar.gz`）。
2. 解压缩下载的文件。
3. 在解压后的目录下，创建一个 `.env` 文件，内容参考上述的环境变量配置部分。
4. 运行可执行文件：

