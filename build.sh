go build -ldflags="-s -w" -o w
pm2 reload w
pm2 logs w