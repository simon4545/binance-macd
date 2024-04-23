go build -ldflags="-s -w" -o bfuture
#upx -9 bfuture
pm2 reload bfuture
pm2 logs bfuture