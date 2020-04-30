.PHONY: build release install test clean

VERSION := 0.1.0

build:
	go build

release:
	mkdir -p release/
	GOOS=linux GOOARCH=386 go build -o release/tfk8s_${VERSION}_linux_386
	zip release/tfk8s_${VERSION}_linux_386.zip release/tfk8s_${VERSION}_linux_386
	GOOS=linux GOOARCH=amd64 go build -o release/tfk8s_${VERSION}_linux_amd64
	zip release/tfk8s_${VERSION}_linux_amd64.zip release/tfk8s_${VERSION}_linux_amd64
	GOOS=linux GOOARCH=arm go build -o release/tfk8s_${VERSION}_linux_arm
	zip release/tfk8s_${VERSION}_linux_arm.zip release/tfk8s_${VERSION}_linux_arm
	GOOS=darwin GOOARCH=amd64 go build -o release/tfk8s_${VERSION}_darwin_amd64
	zip release/tfk8s_${VERSION}_darwin_amd64.zip release/tfk8s_${VERSION}_darwin_amd64
	GOOS=windows GOOARCH=amd64 go build -o release/tfk8s_${VERSION}_windows_amd64
	zip release/tfk8s_${VERSION}_windows_amd64.zip release/tfk8s_${VERSION}_windows_amd64
	GOOS=windows GOOARCH=386 go build -o release/tfk8s_${VERSION}_windows_386
	zip release/tfk8s_${VERSION}_windows_386.zip release/tfk8s_${VERSION}_windows_386

install: build
	go install

test:
	go test 

clean:
	rm -rf release/*
