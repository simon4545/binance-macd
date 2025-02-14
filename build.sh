go build -ldflags="-s -w" -o binance-furture
echo "Build Done"
#upx -9 binancemacd
pm2 reload binance-furture
pm2 logs binance-furture