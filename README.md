## 设计思路

### 数据库

选择`modernc.org/sqlite`数据库，完全go实现，0成本跨平台，简化部署。

#### 数据库表设计

当前使用的 SQLite 数据库包含 3 张核心业务表：`movies`（影片基础信息）、`box_office`（票房信息）和 `ratings`（用户评分）。所有时间字段统一存储为 UTC ISO8601 字符串，外键通过 `PRAGMA foreign_keys = ON` 保证约束生效。

**1. movies 表（影片基础信息）**

存储影片的基础元数据，是其它业务表的主表。

- 主键：`id`（文本）
- 唯一约束：`title`
- 主要用途：作为票房、评分等信息的关联根

字段说明：

| 字段名         | 类型    | 约束 / 说明                                   |
| -------------- | ------- | ---------------------------------------------- |
| `id`           | TEXT    | 主键，影片唯一标识（业务生成的字符串 ID）     |
| `title`        | TEXT    | NOT NULL，UNIQUE，影片名称                    |
| `release_date` | TEXT    | NOT NULL，上映日期（字符串存储，便于索引）    |
| `genre`        | TEXT    | NOT NULL，影片类型 / 风格                     |
| `distributor`  | TEXT    | 发行方，可为空                                |
| `budget`       | INTEGER | 制作成本，单位为货币的基础单位，可为空        |
| `mpa_rating`   | TEXT    | MPA 分级，如 PG-13，可为空                    |
| `created_at`   | TEXT    | NOT NULL，创建时间，默认当前 UTC 时间         |
| `updated_at`   | TEXT    | NOT NULL，更新时间，默认当前 UTC 时间         |

索引：

- `idx_movies_release_date`：基于 `release_date` 的查询（如按上映时间排序 / 过滤）
- `idx_movies_genre`：基于 `genre` 的查询（按类型筛选）
- `idx_movies_created_at`：按创建时间排序或分页

**2. box_office 表（票房信息）**

存储每部影片的票房数据，与 `movies` 为一对一关系。

- 主键：`movie_id`
- 外键：`movie_id` → `movies(id)`，`ON DELETE CASCADE`
- 主要用途：为影片提供全球票房、首周末票房等扩展信息

字段说明：

| 字段名                        | 类型    | 约束 / 说明                                            |
| ----------------------------- | ------- | ------------------------------------------------------- |
| `movie_id`                    | TEXT    | 主键，同时为外键，引用 `movies.id`                     |
| `currency`                    | TEXT    | NOT NULL，货币代码（如 USD、CNY）                      |
| `source`                      | TEXT    | NOT NULL，数据来源标识（如第三方接口、内部统计）       |
| `last_updated`                | TEXT    | NOT NULL，票房数据最后更新时间                         |
| `revenue_worldwide`           | INTEGER | NOT NULL，全球总票房                                   |
| `revenue_opening_weekend_usa` | INTEGER | 美国首周末票房，可为空                                 |

约束与关系：

- 如果某部影片在 `movies` 中被删除，其对应的 `box_office` 记录会被自动级联删除（`ON DELETE CASCADE`），保证数据一致性。
- 每个 `movie_id` 只能有一条票房记录（一对一扩展）。

**3. ratings 表（用户评分）**

存储对影片的评分信息，支持同一影片由多个评分者打分。

- 复合主键：`(movie_id, rater_id)`
- 外键：`movie_id` → `movies(id)`，`ON DELETE CASCADE`
- 主要用途：为影片提供可聚合的评分数据（如平均分、评分人数）

字段说明：

| 字段名      | 类型 | 约束 / 说明                                                    |
| ----------- | ---- | --------------------------------------------------------------- |
| `movie_id`  | TEXT | NOT NULL，外键，引用 `movies.id`                               |
| `rater_id`  | TEXT | NOT NULL，评分者标识（可以是用户 ID、系统 ID 等）              |
| `rating`    | REAL | NOT NULL，评分值，带有 CHECK 约束，范围为 0.5 ≤ rating ≤ 5.0 |
| `updated_at`| TEXT | NOT NULL，评分更新时间，默认当前 UTC 时间                      |

约束与索引：

- 主键 `(movie_id, rater_id)`：同一评分者对同一电影只能保留一条评分记录，重复评分会覆盖（更新）原记录。
- `CHECK (rating >= 0.5 AND rating <= 5.0)`：限制评分值范围，防止非法数据写入。
- 索引 `idx_ratings_movie`：加速按 `movie_id` 查询某部影片的所有评分（例如用于计算平均分）。

### 后端服务

使用`CloudWeGo Hertz`框架，高性能，低延迟，易扩展。  
使用`Swaggo`生成API文档，同时提供OpenAPI3转换工具。

### 优化方向

1. 使用其他高性能数据库（如PostgreSQL）替代SQLite以提升并发处理能力。  
2. 添加内存缓存，减少与数据库交互次数，提高响应速度。


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
