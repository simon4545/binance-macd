go build -ldflags="-s -w" -o w
ssh -i ssdd.pem ubuntu@18.183.90.24 "pm2 stop w"
scp -i ssdd.pem w ubuntu@18.183.90.24:/home/ubuntu/
scp -i ssdd.pem w.yaml ubuntu@18.183.90.24:/home/ubuntu/
ssh -i ssdd.pem ubuntu@18.183.90.24 "pm2 reload w"
# pm2 reload w
# pm2 logs w