go build -ldflags="-s -w" -o bfuture
upx -9 bfuture
# pm2 reload bfuture
# pm2 logs bfuture

ssh root@149.104.27.12 'pm2 stop bfuture;rm -f /root/bfuture/bfuture; '
scp bfuture root@149.104.27.12:/root/bfuture/
ssh root@149.104.27.12 'chmod +x /root/bfuture/bfuture;cd /root/bfuture/; pm2 start bfuture;'