ps -ef | grep 123456 | grep -v grep | awk  '{print $2}' | while read pid; do kill -9  $pid;done
nohup go run main.go -S -l :3001 -c 172.16.0.1/24 -k 123456 -p ws &