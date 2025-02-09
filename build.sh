go build -ldflags="-s -w" -o binance-macd
echo "Build Done"
#upx -9 binancemacd
pm2 reload binance-macd
pm2 logs binance-macd