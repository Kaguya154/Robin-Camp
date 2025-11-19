
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


