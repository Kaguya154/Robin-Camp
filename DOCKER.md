
## Docker

### 构建开发环境镜像

```powershell
docker build -f Dockerfile.dev -t robin-camp-dev .
```

### 使用源码挂载和 Air 热重载启动

```powershell
docker run --rm -it `
  -p 8080:8080 `
  -v ${PWD}:/app `
  -v ${PWD}\.env:/app/.env `
  --name robin-camp-dev `
  robin-camp-dev
```

## 使用 Docker 部署生产环境

### 使用 docker-compose 启动

```powershell
docker compose up -d --build
```

默认会：
- 构建生产镜像（使用多阶段 `Dockerfile`）
- 将宿主机的匿名卷挂载到容器 `/data` 目录存放 SQLite 数据库
- 暴露 `8080` 端口到宿主机

如需自定义环境变量，可编辑 `docker-compose.yml` 中的 `environment` 部分传入。



如果你只想在本地/测试环境直接运行已经发布好的 Docker 镜像（不本地构建），可以参考下面的方式。

### 1. 直接运行已发布 Docker 镜像

假设你在镜像仓库中发布的镜像名为 `YOUR_DOCKER_REPO/robin-camp`，tag 为 `latest` 或某个版本号（例如 `v0.1.1`）：

```powershell
# 直接拉取并运行 latest 版本
$env:AUTH_TOKEN="TOKEN";
$env:DB_URL="file:/data/movies.db?_foreign_keys=on";
$env:BOXOFFICE_URL="https://m1.apifoxmock.com/m1/7149601-6873494-default";
$env:BOXOFFICE_API_KEY="0B4nmUwMPBphsKDr_u9HX";

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
  Kaguya154/robin-camp:latest
```

将 `Kaguya154/robin-camp:latest` 替换为你想要使用的具体 tag（如果不是 latest），例如：

- `Kaguya154/robin-camp:latest`
- `Kaguya154/robin-camp:v0.1.1`

### 2. 使用 docker-compose 运行已发布镜像

如果不希望在本地构建镜像，可以让 `docker-compose.yml` 只使用远端镜像：

```yaml
services:
  app:
    image: docker.io/kaguya154/robin-camp:latest # 实际使用时也可以直接写为 Kaguya154/robin-camp:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - ADDRESS=0.0.0.0
      - AUTH_TOKEN=TOKEN
      - DB_URL=file:/data/movies.db?_foreign_keys=on
      - BOXOFFICE_URL=https://m1.apifoxmock.com/m1/7149601-6873494-default
      - BOXOFFICE_API_KEY=0B4nmUwMPBphsKDr_u9HX
    volumes:
      - movies-data:/data
    restart: unless-stopped

volumes:
  movies-data:
```
