all: 
	go build
	make up

up:
	scp -o StrictHostKeyChecking=no zlogger root@2.2.2.111:~/

clean:
	rm -f zlogger
