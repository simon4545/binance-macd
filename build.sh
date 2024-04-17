go build -ldflags="-s -w" -o binancemacd
upx -9 binancemacd
pm2 reload binancemacd
pm2 logs binancemacd