/**
 * PM2 — YouMind / Minote 服务端
 *
 * 默认进程: notex（与 docker-compose 中 notex 一致，监听 NOTEX_PORT，默认 :8787）
 * Go 程序会从 cwd 加载 .env（pkg/dotenv），请在本文件所在目录保留 .env、config.yaml。
 *
 * 构建:  make build-notex   → bin/notex
 * 启动:  pm2 start ecosystem.config.js
 * 查看:  pm2 logs youmind-notex
 * 停止:  pm2 stop youmind-notex
 *
 * 若要用网关入口（默认 :8080，与 cmd/gateway 一致）:
 *   go build -o bin/gateway ./cmd/gateway
 *   pm2 start ./bin/gateway --name youmind-gateway -i 1 --interpreter none --cwd . --kill-timeout 15000
 * 不要与 notex 同时跑两套 API（默认端口不同但会重复占库/后台任务，除非你很清楚在做什么）。
 */
module.exports = {
  apps: [
    {
      name: 'youmind-notex',
      script: './bin/notex',
      cwd: __dirname,
      interpreter: 'none',
      instances: 1,
      exec_mode: 'fork',
      autorestart: true,
      watch: false,
      max_restarts: 15,
      min_uptime: '10s',
      kill_timeout: 15_000,
      max_memory_restart: '1G',
      time: true,
      env: {
        NODE_ENV: 'production',
      },
    },
  ],
};
