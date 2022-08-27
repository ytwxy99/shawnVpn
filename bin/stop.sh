ps -ef | grep 123456 | grep -v grep | awk  '{print $2}' | while read pid; do kill -9  $pid;done
